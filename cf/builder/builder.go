package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	cf2app "github.com/sclevine/packs/cf2/app"
)

const (
	CodeEnvFailed = iota + 1
	CodeSetupFailed
	CodeBuildFailed
)

func main() {
	app, err := cf2app.New()
	check(err, CodeEnvFailed, "build app env")

	symlink("/app", "/tmp/app")
	symlink("/cache", "/tmp/cache")
	symlink("/lifecycle", "/tmp/lifecycle")
	for _, dir := range []string{
		"/app",
		"/cache",
		"/out",
		"/home/vcap/tmp",
	} {
		err := os.MkdirAll(dir, 0777)
		check(err, CodeSetupFailed, "ensure directory", dir)
		chownAll("vcap", "vcap", dir)
	}

	cmd := exec.Command("/lifecycle/bulider", os.Args[1:]...)
	cmd.Env = append(os.Environ(), app.Stage()...)
	cmd.Dir = "/tmp/app"
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	uid, gid := userLookup("vcap")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid}

	err = cmd.Run()
	check(err, CodeBuildFailed, "failed to build")
}

func symlink(s, t string) {
	if _, err := os.Stat(t); !os.IsNotExist(err) {
		return
	}
	err := os.Symlink(s, t)
	check(err, CodeSetupFailed, "symlink", s, "to", t)
}

func chownAll(user, group, path string) {
	err := exec.Command("chown", "-R", user+":"+group, path).Run()
	check(err, CodeSetupFailed, "chown", path, "to", user+"/"+group)
}

func userLookup(u string) (uid, gid uint32) {
	usr, err := user.Lookup(u)
	check(err, CodeSetupFailed, "find user", u)
	uid64, err := strconv.ParseUint(usr.Uid, 10, 32)
	check(err, CodeSetupFailed, "parse uid", usr.Uid)
	gid64, err := strconv.ParseUint(usr.Gid, 10, 32)
	check(err, CodeSetupFailed, "parse gid", usr.Gid)
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
