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

	herokuapp "github.com/sclevine/packs/heroku/app"
)

const (
	CodeFailedEnv = iota + 1
	CodeFailedSetup
	CodeFailedBuild
	CodeInvalidArgs
)

func main() {
	config := bal.NewLifecycleBuilderConfig(nil, false, false)
	err := config.Parse(os.Args[1:])
	check(err, CodeInvalidArgs, "invalid args")

	var (
		extraArgs      []string
		workingDir     = pwd()
		appZip         = os.Getenv("PACK_APP_ZIP")
		appDir         = config.BuildDir()
		cacheDir       = config.BuildArtifactsCacheDir()
		cacheTar       = config.OutputBuildArtifactsCache()
		cacheTarDir    = filepath.Dir(cacheTar)
		slugletDir     = filepath.Dir(config.OutputDroplet())
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

	ensure(slugletDir, metadataDir, cacheTarDir)
	ensureAll(appDir, cacheDir)
	addBuildpacks("/buildpacks", buildpacksDir)

	if strings.Join(buildpackOrder, "") == "" && !skipDetect {
		extraArgs = append(extraArgs, "-buildpackOrder", reduceJSONFile("name", buildpackConf))
	}

	uid, gid := userLookup("heroku")
	setupEnv()

	cmd := exec.Command("/lifecycle/builder", append(os.Args[1:], extraArgs...)...)
	cmd.Dir = appDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}

	err = cmd.Run()
	check(err, CodeFailedBuild, "build")
}

func copyAppDir(src, dst string) {
	copier := appfiles.ApplicationFiles{}
	files, err := copier.AppFilesInDir(src)
	check(err, CodeFailedSetup, "analyze app")
	err = copier.CopyFiles(files, src, dst)
	check(err, CodeFailedSetup, "copy app")
}

func copyAppZip(src, dst string) {
	zipper := appfiles.ApplicationZipper{}
	tmpDir, err := ioutil.TempDir("", "pack")
	check(err, CodeFailedSetup, "create temp dir")
	defer os.RemoveAll(tmpDir)
	err = zipper.Unzip(src, tmpDir)
	check(err, CodeFailedSetup, "unzip app")
	copyAppDir(tmpDir, dst)
}

func ensure(dirs ...string) {
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0777)
		check(err, CodeFailedSetup, "make directory", dir)
		err = exec.Command("chown", "heroku:heroku", dir).Run()
		check(err, CodeFailedSetup, "chown", dir, "to heroku:heroku")
	}
}

func ensureAll(dirs ...string) {
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0777)
		check(err, CodeFailedSetup, "make directory", dir)
		err = exec.Command("chown", "-R", "heroku:heroku", dir).Run()
		check(err, CodeFailedSetup, "recursively chown", dir, "to", "heroku:heroku")
	}
}

func addBuildpacks(src, dst string) {
	files, err := ioutil.ReadDir(src)
	if os.IsNotExist(err) {
		return
	}
	check(err, CodeFailedSetup, "setup buildpacks", src)

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
	check(err, CodeFailedSetup, "open", path)
	var list []map[string]string
	err = json.NewDecoder(f).Decode(&list)
	check(err, CodeFailedSetup, "decode", path)

	var out []string
	for _, m := range list {
		out = append(out, m[key])
	}
	return strings.Join(out, ",")
}

func setupEnv() {
	app, err := herokuapp.New()
	check(err, CodeFailedEnv, "build app env")
	for k, v := range app.Stage() {
		err := os.Setenv(k, v)
		check(err, CodeFailedEnv, "set app env")
	}
}

func unzip(zip, dst string) {
	err := os.MkdirAll(dst, 0777)
	check(err, CodeFailedSetup, "ensure directory", dst)
	err = exec.Command("unzip", "-qq", zip, "-d", dst).Run()
	check(err, CodeFailedSetup, "unzip", zip, "to", dst)
}

func untar(tar, dst string) {
	err := os.MkdirAll(dst, 0777)
	check(err, CodeFailedSetup, "ensure directory", dst)
	err = exec.Command("tar", "-C", dst, "-xzf", tar).Run()
	check(err, CodeFailedSetup, "untar", tar, "to", dst)
}

func pwd() string {
	wd, err := os.Getwd()
	check(err, CodeFailedSetup, "get working directory")
	return wd
}

func userLookup(u string) (uid, gid uint32) {
	usr, err := user.Lookup(u)
	check(err, CodeFailedSetup, "find user", u)
	uid64, err := strconv.ParseUint(usr.Uid, 10, 32)
	check(err, CodeFailedSetup, "parse uid", usr.Uid)
	gid64, err := strconv.ParseUint(usr.Gid, 10, 32)
	check(err, CodeFailedSetup, "parse gid", usr.Gid)
	return uint32(uid64), uint32(gid64)
}

func check(err error, code int, action ...string) {
	if err == nil {
		return
	}
	message := "failed to " + strings.Join(action, " ")
	fmt.Fprintf(os.Stderr, "Error: %s: %s", message, err)
	os.Exit(code)
}
