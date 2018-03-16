package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	herokuapp "github.com/sclevine/packs/heroku/app"
)

const (
	CodeFailedEnv = iota + 1
	CodeFailedSetup
	CodeFailedShell
)

func main() {
	err := os.Chdir("/app")
	check(err, CodeFailedSetup, "change directory")

	app, err := herokuapp.New()
	check(err, CodeFailedEnv, "build app env")
	for k, v := range app.Launch() {
		err := os.Setenv(k, v)
		check(err, CodeFailedEnv, "set app env")
	}

	args := append([]string{"/lifecycle/shell", "/app"}, os.Args[1:]...)
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
