package main

import (
	"encoding/json"
	"flag"
	"os"
	"strings"
	"syscall"

	cfapp "github.com/sclevine/packs/cf/app"
	"github.com/sclevine/packs/cf/sys"
)

const (
	homeDir         = "/home/vcap"
	appDir          = "/home/vcap/app"
	stagingInfoFile = "/home/vcap/staging_info.yml"
)

func main() {
	defer sys.Cleanup()

	var droplet string
	flag.StringVar(&droplet, "droplet", os.Getenv("PACK_DROPLET_PATH"), "file containing compressed droplet")
	flag.Parse()

	command := strings.Join(flag.Args(), " ")

	if err := supplyApp(droplet, homeDir); err != nil {
		sys.Fatal(err, sys.CodeFailed, "supply app")
	}

	if err := os.Chdir(appDir); err != nil {
		sys.Fatal(err, sys.CodeFailed, "change directory to", appDir)
	}

	if command == "" {
		var err error
		command, err = readCommand(stagingInfoFile)
		if err != nil {
			sys.Fatal(err, sys.CodeFailed, "determine start command")
		}
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

	args := []string{"/lifecycle/launcher", appDir, command, ""}
	if err := syscall.Exec("/lifecycle/launcher", args, os.Environ()); err != nil {
		sys.Fatal(err, sys.CodeFailedLaunch, "launch")
	}
}

func supplyApp(tgz, dst string) error {
	if _, err := os.Stat(tgz); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return sys.Fail(err, "stat", tgz)
	}
	if _, err := sys.Run("tar", "-C", dst, "-xzf", tgz); err != nil {
		return sys.Fail(err, "untar", tgz, "to", dst)
	}
	return nil
}

func readCommand(path string) (string, error) {
	stagingInfo, err := os.Open(path)
	if err != nil {
		return "", sys.Fail(err, "read start command")
	}
	var info struct {
		StartCommand string `json:"start_command"`
	}
	if err := json.NewDecoder(stagingInfo).Decode(&info); err != nil {
		return "", sys.Fail(err, "parse start command")
	}
	return info.StartCommand, nil
}