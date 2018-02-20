package main

import (
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
	err := config.Parse(os.Args[1:])
	check(err, CodeInvalidArgs, "invalid args")

	var (
		appDir      = config.BuildDir()
		cacheDir    = config.BuildArtifactsCacheDir()
		dropletDir  = filepath.Dir(config.OutputDroplet())
		metadataDir = filepath.Dir(config.OutputMetadata())
	)

	ensureDirs(appDir, cacheDir, dropletDir, metadataDir, "/home/vcap/tmp")
	ensureLink(appDir, "/tmp/app")
	ensureLink(cacheDir, "/tmp/cache")

	setupBuildpacks("/buildpacks", config.BuildpackPath)

	app, err := cfapp.New()
	check(err, CodeFailedEnv, "build app env")

	cmd := exec.Command("/lifecycle/builder", os.Args[1:]...)
	cmd.Env = append(os.Environ(), app.Stage()...)
	cmd.Dir = appDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	uid, gid := userLookup("vcap")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid}

	err = cmd.Run()
	check(err, CodeFailedBuild, "build")
}

func ensureDirs(dirs ...string) {
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0777)
		check(err, CodeFailedSetup, "ensure directory", dir)
		chownAll("vcap", "vcap", dir)
	}
}

func ensureLink(s, t string) {
	if _, err := os.Stat(t); !os.IsNotExist(err) {
		return
	}
	err := os.Symlink(s, t)
	check(err, CodeFailedSetup, "symlink", s, "to", t)
}

func setupBuildpacks(dir string, dest func(string) string) {
	files, err := ioutil.ReadDir(dir)
	if os.IsNotExist(err) {
		return
	}
	check(err, CodeFailedSetup, "setup buildpacks", dir)

	for _, f := range files {
		filename := f.Name()
		ext := filepath.Ext(filename)
		if strings.ToLower(ext) != ".zip" {
			continue
		}
		zip := filepath.Join(dir, filename)
		dst := dest(strings.TrimSuffix(filename, ext))
		unzip(zip, dst)
	}
}

func unzip(zip, dst string) {
	err := os.MkdirAll(dst, 0777)
	check(err, CodeFailedSetup, "ensure directory", dst)
	err = exec.Command("unzip", "-qq", zip, "-d", dst).Run()
	check(err, CodeFailedSetup, "unzip", zip, "to", dst)
}

func chownAll(user, group, path string) {
	err := exec.Command("chown", "-R", user+":"+group, path).Run()
	check(err, CodeFailedSetup, "chown", path, "to", user+"/"+group)
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
