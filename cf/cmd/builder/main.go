package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/sclevine/packs/cf/build"
	"github.com/sclevine/packs/cf/sys"
)

func main() {
	defer sys.Cleanup()

	config := bal.NewLifecycleBuilderConfig(nil, false, false)
	if err := config.Parse(os.Args[1:]); err != nil {
		sys.Fatal(err, sys.CodeInvalidArgs, "parse arguments")
	}

	var (
		extraArgs  []string
		appVersion string

		appName = os.Getenv("PACK_APP_NAME")
		appZip  = os.Getenv("PACK_APP_ZIP")
		appDir  = os.Getenv("PACK_APP_DIR")

		buildDir     = config.BuildDir()
		cacheDir     = config.BuildArtifactsCacheDir()
		cacheTar     = config.OutputBuildArtifactsCache()
		metadataJSON = config.OutputMetadata()

		cacheTarDir = filepath.Dir(cacheTar)
		metadataDir = filepath.Dir(metadataJSON)
		dropletDir  = filepath.Dir(config.OutputDroplet())

		buildpacksDir  = config.BuildpacksDir()
		buildpackConf  = filepath.Join(buildpacksDir, "config.json")
		buildpackOrder = config.BuildpackOrder()
		skipDetect     = config.SkipDetect()
	)

	if appDir == "" {
		var err error
		if appDir, err = os.Getwd(); err != nil {
			sys.Fatal(err, sys.CodeFailed, "get working directory")
		}
	}

	if appZip != "" {
		appVersion = fileSHA(appZip)
		if err := copyAppZip(appZip, buildDir); err != nil {
			sys.Fatal(err, sys.CodeFailed, "extract app zip")
		}
	} else if appDir != "" {
		appVersion = commitSHA(appDir)
		if !cmpDir(appDir, buildDir) {
			if err := copyAppDir(appDir, buildDir); err != nil {
				sys.Fatal(err, sys.CodeFailed, "copy app directory")
			}
		}
	} else {
		sys.Fatal(nil, sys.CodeInvalidArgs, "parse app directory")
	}

	if _, err := os.Stat(cacheTar); err == nil {
		if err := untar(cacheTar, cacheDir); err != nil {
			sys.Fatal(err, sys.CodeFailed, "extract cache")
		}
	}

	if err := mkdir(dropletDir, metadataDir, cacheTarDir); err != nil {
		sys.Fatal(err, sys.CodeFailed, "prepare destination directories")
	}
	if err := mkdirAll(buildDir, cacheDir, "/home/vcap/tmp"); err != nil {
		sys.Fatal(err, sys.CodeFailed, "prepare source directories")
	}
	if err := addBuildpacks("/buildpacks", buildpacksDir); err != nil {
		sys.Fatal(err, sys.CodeFailed, "add buildpacks")
	}

	if strings.Join(buildpackOrder, "") == "" && !skipDetect {
		names, err := reduceJSONFile("name", buildpackConf)
		if err != nil {
			sys.Fatal(err, sys.CodeFailed, "determine buildpack names")
		}
		extraArgs = append(extraArgs, "-buildpackOrder", names)
	}

	uid, gid, err := userLookup("vcap")
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "determine vcap UID/GID")
	}
	if err := setupStdFds(); err != nil {
		sys.Fatal(err, sys.CodeFailed, "setup fds")
	}
	if err := setupEnv(); err != nil {
		sys.Fatal(err, sys.CodeInvalidEnv, "setup env")
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
		sys.Fatal(err, sys.CodeFailedBuild, "build")
	}
	if err := setKeyJSON(metadataJSON, "pack_metadata", build.PackMetadata{
		App: build.AppMetadata{
			Name:    appName,
			Version: appVersion,
		},
	}); err != nil {
		sys.Fatal(err, sys.CodeFailed, "write metadata")
	}
}

func copyAppDir(src, dst string) error {
	copier := appfiles.ApplicationFiles{}
	files, err := copier.AppFilesInDir(src)
	if err != nil {
		return sys.Fail(err, "analyze app in", src)
	}
	if err := copier.CopyFiles(files, src, dst); err != nil {
		return sys.Fail(err, "copy app from", src, "to", dst)
	}
	return nil
}

func copyAppZip(src, dst string) error {
	zipper := appfiles.ApplicationZipper{}
	tmpDir, err := ioutil.TempDir("", "pack")
	if err != nil {
		return sys.Fail(err, "create temp dir")
	}
	defer os.RemoveAll(tmpDir)
	if err := zipper.Unzip(src, tmpDir); err != nil {
		return sys.Fail(err, "unzip app from", src, "to", tmpDir)
	}
	return copyAppDir(tmpDir, dst)
}

func cmpDir(dirs ...string) bool {
	var last string
	for _, dir := range dirs {
		next, err := filepath.Abs(dir)
		if err != nil {
			return false
		}
		switch last {
		case "", next:
			last = next
		default:
			return false
		}
	}
	return true
}

func mkdir(dirs ...string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return sys.Fail(err, "make directory", dir)
		}
		if _, err := sys.Run("chown", "vcap:vcap", dir); err != nil {
			return sys.Fail(err, "chown", dir, "to vcap:vcap")
		}
	}
	return nil
}

func mkdirAll(dirs ...string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return sys.Fail(err, "make directory", dir)
		}
		if _, err := sys.Run("chown", "-R", "vcap:vcap", dir); err != nil {
			return sys.Fail(err, "recursively chown", dir, "to", "vcap:vcap")
		}
	}
	return nil
}

func commitSHA(dir string) string {
	v, err := sys.Run("git", "-C", dir, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return v
}

func fileSHA(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// TODO: test with /dev/null
func setKeyJSON(path, key string, value interface{}) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return sys.Fail(err, "open metadata")
	}
	defer f.Close()

	var contents map[string]interface{}
	if err := json.NewDecoder(f).Decode(&contents); err != nil {
		return sys.Fail(err, "decode JSON at", path)
	}
	contents[key] = value
	if _, err := f.Seek(0, 0); err != nil {
		return sys.Fail(err, "seek file at", path)
	}
	if err := f.Truncate(0); err != nil {
		return sys.Fail(err, "truncate file at", path)
	}
	if err := json.NewEncoder(f).Encode(contents); err != nil {
		return sys.Fail(err, "encode JSON to", path)
	}
	return nil
}

func addBuildpacks(src, dst string) error {
	files, err := ioutil.ReadDir(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return sys.Fail(err, "setup buildpacks", src)
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
		return "", sys.Fail(err, "open", path)
	}
	var list []map[string]string
	if err := json.NewDecoder(f).Decode(&list); err != nil {
		return "", sys.Fail(err, "decode", path)
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
		return sys.Fail(err, "build app env")
	}
	for k, v := range app.Stage() {
		err := os.Setenv(k, v)
		if err != nil {
			return sys.Fail(err, "set app env")
		}
	}
	return nil
}

func setupStdFds() error {
	cmd := exec.Command("chown", "vcap", "/dev/stdout", "/dev/stderr")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return sys.Fail(err, "fix permissions of stdout and stderr")
	}
	return nil
}

func unzip(zip, dst string) error {
	if err := os.MkdirAll(dst, 0777); err != nil {
		return sys.Fail(err, "ensure directory", dst)
	}
	if _, err := sys.Run("unzip", "-qq", zip, "-d", dst); err != nil {
		return sys.Fail(err, "unzip", zip, "to", dst)
	}
	return nil
}

func untar(tar, dst string) error {
	if err := os.MkdirAll(dst, 0777); err != nil {
		return sys.Fail(err, "ensure directory", dst)
	}
	if _, err := sys.Run("tar", "-C", dst, "-xzf", tar); err != nil {
		return sys.Fail(err, "untar", tar, "to", dst)
	}
	return nil
}

func userLookup(u string) (uid, gid uint32, err error) {
	usr, err := user.Lookup(u)
	if err != nil {
		return 0, 0, sys.Fail(err, "find user", u)
	}
	uid64, err := strconv.ParseUint(usr.Uid, 10, 32)
	if err != nil {
		return 0, 0, sys.Fail(err, "parse uid", usr.Uid)
	}
	gid64, err := strconv.ParseUint(usr.Gid, 10, 32)
	if err != nil {
		return 0, 0, sys.Fail(err, "parse gid", usr.Gid)
	}
	return uint32(uid64), uint32(gid64), nil
}
