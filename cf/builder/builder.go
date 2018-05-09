package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	bal "code.cloudfoundry.org/buildpackapplifecycle"
	"code.cloudfoundry.org/cli/cf/appfiles"

	"bytes"

	cfapp "github.com/sclevine/packs/cf/app"
)

const (
	CodeFailedEnv = iota + 1
	CodeFailedSetup
	CodeFailedBuild
	CodeInvalidArgs
)

func main() {
	config := bal.NewLifecycleBuilderConfig(nil, false, false)
	if err := config.Parse(os.Args[1:]); err != nil {
		fatal(err, CodeInvalidArgs, "invalid args")
	}

	var (
		extraArgs      []string
		appZip         = os.Getenv("PACK_APP_ZIP")
		appDir         = os.Getenv("PACK_APP_DIR")
		buildDir       = config.BuildDir()
		cacheDir       = config.BuildArtifactsCacheDir()
		cacheTar       = config.OutputBuildArtifactsCache()
		cacheTarDir    = filepath.Dir(cacheTar)
		dropletDir     = filepath.Dir(config.OutputDroplet())
		metadataDir    = filepath.Dir(config.OutputMetadata())
		buildpacksDir  = config.BuildpacksDir()
		buildpackConf  = filepath.Join(buildpacksDir, "config.json")
		buildpackOrder = config.BuildpackOrder()
		skipDetect     = config.SkipDetect()
	)

	if appDir == "" {
		var err error
		if appDir, err = os.Getwd(); err != nil {
			fatal(err, CodeFailedSetup, "get working directory")
		}
	}

	if appZip != "" {
		if err := copyAppZip(appZip, buildDir); err != nil {
			fatal(err, CodeFailedSetup, "extract app zip")
		}
	} else if appDir != buildDir {
		if err := copyAppDir(appDir, buildDir); err != nil {
			fatal(err, CodeFailedSetup, "copy app directory")
		}
	}

	if _, err := os.Stat(cacheTar); err == nil {
		if err := untar(cacheTar, cacheDir); err != nil {
			fatal(err, CodeFailedSetup, "extract cache")
		}
	}

	if err := ensure(dropletDir, metadataDir, cacheTarDir); err != nil {
		fatal(err, CodeFailedSetup, "prepare destination directories")
	}
	if err := ensureAll(buildDir, cacheDir, "/home/vcap/tmp"); err != nil {
		fatal(err, CodeFailedSetup, "prepare source directories")
	}
	if err := addBuildpacks("/buildpacks", buildpacksDir); err != nil {
		fatal(err, CodeFailedSetup, "add buildpacks")
	}

	if strings.Join(buildpackOrder, "") == "" && !skipDetect {
		names, err := reduceJSONFile("name", buildpackConf)
		if err != nil {
			fatal(err, CodeFailedSetup, "determine buildpack names")
		}
		extraArgs = append(extraArgs, "-buildpackOrder", names)
	}

	uid, gid, err := userLookup("vcap")
	if err != nil {
		fatal(err, CodeFailedSetup, "determine vcap UID/GID")
	}
	if err := setupStdFds(); err != nil {
		fatal(err, CodeFailedSetup, "setup fds")
	}
	if err := setupEnv(); err != nil {
		fatal(err, CodeFailedEnv, "setup env")
	}

	cmd := exec.Command("/lifecycle/builder", append(os.Args[1:], extraArgs...)...)
	cmd.Dir = buildDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}
	if err := cmd.Run(); err != nil {
		fatal(err, CodeFailedBuild, "build")
	}
}

func copyAppDir(src, dst string) error {
	copier := appfiles.ApplicationFiles{}
	files, err := copier.AppFilesInDir(src)
	if err != nil {
		return fail(err, "analyze app in", src)
	}
	err = copier.CopyFiles(files, src, dst)
	if err != nil {
		return fail(err, "copy app from", src, "to", dst)
	}
	return nil
}

func copyAppZip(src, dst string) error {
	zipper := appfiles.ApplicationZipper{}
	tmpDir, err := ioutil.TempDir("", "pack")
	if err != nil {
		return fail(err, "create temp dir")
	}
	defer os.RemoveAll(tmpDir)
	if err := zipper.Unzip(src, tmpDir); err != nil {
		return fail(err, "unzip app from", src, "to", tmpDir)
	}
	return copyAppDir(tmpDir, dst)
}

func ensure(dirs ...string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return fail(err, "make directory", dir)
		}
		if err := runCmd("chown", "vcap:vcap", dir); err != nil {
			return fail(err, "chown", dir, "to vcap:vcap")
		}
	}
	return nil
}

func ensureAll(dirs ...string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return fail(err, "make directory", dir)
		}
		if err := runCmd("chown", "-R", "vcap:vcap", dir); err != nil {
			return fail(err, "recursively chown", dir, "to", "vcap:vcap")
		}
	}
	return nil
}

func addBuildpacks(src, dst string) error {
	files, err := ioutil.ReadDir(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fail(err, "setup buildpacks", src)
	}

	for _, f := range files {
		filename := f.Name()
		ext := filepath.Ext(filename)
		if strings.ToLower(ext) != ".zip" || len(filename) != 36 {
			continue
		}
		sum := strings.ToLower(strings.TrimSuffix(filename, ext))
		unzip(filepath.Join(src, filename), filepath.Join(dst, sum))
	}
	return nil
}

func reduceJSONFile(key string, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fail(err, "open", path)
	}
	var list []map[string]string
	if err := json.NewDecoder(f).Decode(&list); err != nil {
		return "", fail(err, "decode", path)
	}

	var out []string
	for _, m := range list {
		out = append(out, m[key])
	}
	return strings.Join(out, ","), nil
}

func setupEnv() error {
	app, err := cfapp.New()
	if err != nil {
		return fail(err, "build app env")
	}
	for k, v := range app.Stage() {
		err := os.Setenv(k, v)
		if err != nil {
			return fail(err, "set app env")
		}
	}
	return nil
}

func setupStdFds() error {
	cmd := exec.Command("chown", "vcap", "/dev/stdout", "/dev/stderr")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fail(err, "fix permissions of stdout and stderr")
	}
	return nil
}

func unzip(zip, dst string) error {
	if err := os.MkdirAll(dst, 0777); err != nil {
		return fail(err, "ensure directory", dst)
	}
	if err := runCmd("unzip", "-qq", zip, "-d", dst); err != nil {
		return fail(err, "unzip", zip, "to", dst)
	}
	return nil
}

func untar(tar, dst string) error {
	if err := os.MkdirAll(dst, 0777); err != nil {
		return fail(err, "ensure directory", dst)
	}
	if err := runCmd("tar", "-C", dst, "-xzf", tar); err != nil {
		return fail(err, "untar", tar, "to", dst)
	}
	return nil
}

func userLookup(u string) (uid, gid uint32, err error) {
	usr, err := user.Lookup(u)
	if err != nil {
		return 0, 0, fail(err, "find user", u)
	}
	uid64, err := strconv.ParseUint(usr.Uid, 10, 32)
	if err != nil {
		return 0, 0, fail(err, "parse uid", usr.Uid)
	}
	gid64, err := strconv.ParseUint(usr.Gid, 10, 32)
	if err != nil {
		return 0, 0, fail(err, "parse gid", usr.Gid)
	}
	return uint32(uid64), uint32(gid64), nil
}

func runCmd(name string, arg ...string) error {
	buf := &bytes.Buffer{}
	cmd := exec.Command(name, arg...)
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %s\n%s", name, err, buf.String())
	}
}

func fail(err error, action ...string) error {
	message := "failed to " + strings.Join(action, " ")
	return fmt.Errorf("%s: %s", message, err)
}

func fatal(err error, code int, action ...string) {
	message := "failed to " + strings.Join(action, " ")
	fmt.Fprintf(os.Stderr, "Error: %s: %s", message, err)
	os.Exit(code)
}
