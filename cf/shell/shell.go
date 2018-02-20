package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	cfapp "github.com/sclevine/packs/cf/app"
)

const (
	CodeFailedEnv = iota + 1
	CodeFailedSetup
	CodeFailedShell
)

func main() {
	err := os.Chdir("/home/vcap/app")
	check(err, CodeFailedSetup, "change directory")

	app, err := cfapp.New()
	check(err, CodeFailedEnv, "build app env")
	for k, v := range app.Launch() {
		err := os.Setenv(k, v)
		check(err, CodeFailedEnv, "set app env")
	}

	args := append([]string{"/lifecycle/shell", "/home/vcap/app"}, os.Args[1:]...)
	err = syscall.Exec("/lifecycle/shell", args, os.Environ())
	check(err, CodeFailedShell, "run")
}

func check(err error, code int, action ...string) {
	if err == nil {
		return
	}
	message := "failed to " + strings.Join(action, " ")
	fmt.Fprintf(os.Stderr, "Error: %s: %s", message, err)
	os.Exit(code)
}
