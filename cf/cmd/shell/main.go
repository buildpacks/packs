package main

import (
	"os"
	"syscall"

	cfapp "github.com/sclevine/packs/cf/app"
	"github.com/sclevine/packs/cf/sys"
)

const appDir = "/home/vcap/app"

func main() {
	defer sys.Cleanup()

	if err := os.Chdir(appDir); err != nil {
		sys.Fatal(err, sys.CodeFailed, "change directory")
	}
	app, err := cfapp.New()
	if err != nil {
		sys.Fatal(err, sys.CodeInvalidEnv, "build app env")
	}
	for k, v := range app.Launch() {
		if err := os.Setenv(k, v); err != nil {
			sys.Fatal(err, sys.CodeInvalidEnv, "set app env")
		}
	}
	args := append([]string{"/lifecycle/shell", appDir}, os.Args[1:]...)
	if err := syscall.Exec("/lifecycle/shell", args, os.Environ()); err != nil {
		sys.Fatal(err, sys.CodeFailedLaunch, "run")
	}
}
