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
	err := config.Parse(os.Args[1:])
	check(err, CodeInvalidArgs, "invalid args")

	var (
		appDir         = config.BuildDir()
		cacheDir       = config.BuildArtifactsCacheDir()
		dropletDir     = filepath.Dir(config.OutputDroplet())
		metadataDir    = filepath.Dir(config.OutputMetadata())
		buildpackConf  = filepath.Join(config.BuildpacksDir(), "config.json")
		buildpackOrder = config.BuildpackOrder()
	)

	copyApp(".", appDir)
	ensureDirs(appDir, cacheDir, dropletDir, metadataDir, "/home/vcap/tmp")
	ensureLink(appDir, "/tmp/app")
	ensureLink(cacheDir, "/tmp/cache")
	addBuildpacks("/buildpacks", config.BuildpackPath)

	var extraArgs []string
	if strings.Join(buildpackOrder, "") == "" {
		extraArgs = append(extraArgs, "-buildpackOrder", reduceJSONFile("name", buildpackConf))
	}

	uid, gid := userLookup("vcap")
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

func copyApp(src, dst string) {
	copier := appfiles.ApplicationFiles{}
	files, err := copier.AppFilesInDir(src)
	check(err, CodeFailedSetup, "analyze app")
	err = copier.CopyFiles(files, src, dst)
	check(err, CodeFailedSetup, "copy app")
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

func addBuildpacks(dir string, dest func(string) string) {
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
	app, err := cfapp.New()
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
