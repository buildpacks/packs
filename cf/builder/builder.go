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
		workingDir     = pwd()
		appZip         = os.Getenv("PACK_APP_ZIP")
		appDir         = config.BuildDir()
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

	if appZip != "" {
		copyAppZip(appZip, appDir)
	} else if appDir != workingDir {
		copyAppDir(workingDir, appDir)
	}

	if _, err := os.Stat(cacheTar); err == nil {
		untar(cacheTar, cacheDir)
	}

	ensure(dropletDir, metadataDir, cacheTarDir)
	ensureAll(appDir, cacheDir, "/home/vcap/tmp")
	addBuildpacks("/buildpacks", buildpacksDir)

	if strings.Join(buildpackOrder, "") == "" && !skipDetect {
		extraArgs = append(extraArgs, "-buildpackOrder", reduceJSONFile("name", buildpackConf))
	}

	uid, gid := userLookup("vcap")
	setupStdFds()
	setupEnv()

	cmd := exec.Command("/lifecycle/builder", append(os.Args[1:], extraArgs...)...)
	cmd.Dir = appDir
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

func copyAppDir(src, dst string) {
	copier := appfiles.ApplicationFiles{}
	files, err := copier.AppFilesInDir(src)
	if err != nil {
		fatal(err, CodeFailedSetup, "analyze app")
	}
	err = copier.CopyFiles(files, src, dst)
	if err != nil {
		fatal(err, CodeFailedSetup, "copy app")
	}
}

func copyAppZip(src, dst string) {
	zipper := appfiles.ApplicationZipper{}
	tmpDir, err := ioutil.TempDir("", "pack")
	if err != nil {
		fatal(err, CodeFailedSetup, "create temp dir")
	}
	defer os.RemoveAll(tmpDir)
	if err := zipper.Unzip(src, tmpDir); err != nil {
		fatal(err, CodeFailedSetup, "unzip app")
	}
	copyAppDir(tmpDir, dst)
}

func ensure(dirs ...string) {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0777); err != nil {
			fatal(err, CodeFailedSetup, "make directory", dir)
		}
		if err := exec.Command("chown", "vcap:vcap", dir).Run(); err != nil {
			fatal(err, CodeFailedSetup, "chown", dir, "to vcap:vcap")
		}
	}
}

func ensureAll(dirs ...string) {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0777); err != nil {
			fatal(err, CodeFailedSetup, "make directory", dir)
		}
		if err := exec.Command("chown", "-R", "vcap:vcap", dir).Run(); err != nil {
			fatal(err, CodeFailedSetup, "recursively chown", dir, "to", "vcap:vcap")
		}
	}
}

func addBuildpacks(src, dst string) {
	files, err := ioutil.ReadDir(src)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		fatal(err, CodeFailedSetup, "setup buildpacks", src)
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
}

func reduceJSONFile(key string, path string) string {
	f, err := os.Open(path)
	if err != nil {
		fatal(err, CodeFailedSetup, "open", path)
	}
	var list []map[string]string
	if err := json.NewDecoder(f).Decode(&list); err != nil {
		fatal(err, CodeFailedSetup, "decode", path)
	}

	var out []string
	for _, m := range list {
		out = append(out, m[key])
	}
	return strings.Join(out, ",")
}

func setupEnv() {
	app, err := cfapp.New()
	if err != nil {
		fatal(err, CodeFailedEnv, "build app env")
	}
	for k, v := range app.Stage() {
		err := os.Setenv(k, v)
		if err != nil {
			fatal(err, CodeFailedEnv, "set app env")
		}
	}
}

func setupStdFds() {
	cmd := exec.Command("chown", "vcap", "/dev/stdout", "/dev/stderr")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal(err, CodeFailedSetup, "fix permissions of stdout and stderr")
	}
}

func unzip(zip, dst string) {
	if err := os.MkdirAll(dst, 0777); err != nil {
		fatal(err, CodeFailedSetup, "ensure directory", dst)
	}
	if err := exec.Command("unzip", "-qq", zip, "-d", dst).Run(); err != nil {
		fatal(err, CodeFailedSetup, "unzip", zip, "to", dst)
	}
}

func untar(tar, dst string) {
	if err := os.MkdirAll(dst, 0777); err != nil {
		fatal(err, CodeFailedSetup, "ensure directory", dst)
	}
	if err := exec.Command("tar", "-C", dst, "-xzf", tar).Run(); err != nil {
		fatal(err, CodeFailedSetup, "untar", tar, "to", dst)
	}
}

func pwd() string {
	wd, err := os.Getwd()
	if err != nil {
		fatal(err, CodeFailedSetup, "get working directory")
	}
	return wd
}

func userLookup(u string) (uid, gid uint32) {
	usr, err := user.Lookup(u)
	if err != nil {
		fatal(err, CodeFailedSetup, "find user", u)
	}
	uid64, err := strconv.ParseUint(usr.Uid, 10, 32)
	if err != nil {
		fatal(err, CodeFailedSetup, "parse uid", usr.Uid)
	}
	gid64, err := strconv.ParseUint(usr.Gid, 10, 32)
	if err != nil {
		fatal(err, CodeFailedSetup, "parse gid", usr.Gid)
	}
	return uint32(uid64), uint32(gid64)
}

func fatal(err error, code int, action ...string) {
	message := "failed to " + strings.Join(action, " ")
	fmt.Fprintf(os.Stderr, "Error: %s: %s", message, err)
	os.Exit(code)
}
