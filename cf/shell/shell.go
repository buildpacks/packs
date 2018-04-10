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
	if err := os.Chdir("/home/vcap/app"); err != nil {
		fatal(err, CodeFailedSetup, "change directory")
	}
	app, err := cfapp.New()
	if err != nil {
		fatal(err, CodeFailedEnv, "build app env")
	}
	for k, v := range app.Launch() {
		if err := os.Setenv(k, v); err != nil {
			fatal(err, CodeFailedEnv, "set app env")
		}
	}

	args := append([]string{"/lifecycle/shell", "/home/vcap/app"}, os.Args[1:]...)
	if err := syscall.Exec("/lifecycle/shell", args, os.Environ()); err != nil {
		fatal(err, CodeFailedShell, "run")
	}
}

func fatal(err error, code int, action ...string) {
	message := "failed to " + strings.Join(action, " ")
	fmt.Fprintf(os.Stderr, "Error: %s: %s", message, err)
	os.Exit(code)
}
