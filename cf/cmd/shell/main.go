package main

import (
	"os"
	"syscall"

	cfapp "github.com/sclevine/packs/cf/app"
	"github.com/sclevine/packs/cf/sys"
)

const appDir = "/home/vcap/app"

func main() {
	sys.Exit(shell())
}

func shell() error {
	if err := os.Chdir(appDir); err != nil {
		return sys.FailErr(err, "change directory")
	}
	app, err := cfapp.New()
	if err != nil {
		return sys.FailErrCode(err, sys.CodeInvalidEnv, "build app env")
	}
	for k, v := range app.Launch() {
		if err := os.Setenv(k, v); err != nil {
			return sys.FailErrCode(err, sys.CodeInvalidEnv, "set app env")
		}
	}
	args := append([]string{"/lifecycle/shell", appDir}, os.Args[1:]...)
	if err := syscall.Exec("/lifecycle/shell", args, os.Environ()); err != nil {
		return sys.FailErrCode(err, sys.CodeFailedLaunch, "run")
	}
	return nil
}
