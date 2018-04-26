package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	cfapp "github.com/sclevine/packs/cf/app"
)

const (
	CodeFailedEnv = iota + 1
	CodeFailedSetup
	CodeFailedLaunch
)

func main() {
	var inputDroplet string
	flag.StringVar(&inputDroplet, "inputDroplet", "/tmp/droplet", "file containing compressed droplet")
	flag.Parse()
	command := strings.Join(flag.Args(), " ")

	supplyApp(inputDroplet, "/home/vcap")

	if err := os.Chdir("/home/vcap/app"); err != nil {
		fatal(err, CodeFailedSetup, "change directory")
	}

	if command == "" {
		command = readCommand("/home/vcap/staging_info.yml")
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

	args := []string{"/lifecycle/launcher", "/home/vcap/app", command, ""}
	if err := syscall.Exec("/lifecycle/launcher", args, os.Environ()); err != nil {
		fatal(err, CodeFailedLaunch, "launch")
	}
}

func supplyApp(tgz, dst string) {
	if _, err := os.Stat(tgz); os.IsNotExist(err) {
		return
	} else if err != nil {
		fatal(err, CodeFailedSetup, "stat", tgz)
	}
	if err := exec.Command("tar", "-C", dst, "-xzf", tgz).Run(); err != nil {
		fatal(err, CodeFailedSetup, "untar", tgz, "to", dst)
	}
}

func readCommand(path string) string {
	stagingInfo, err := os.Open(path)
	if err != nil {
		fatal(err, CodeFailedSetup, "read start command")
	}
	var info struct {
		StartCommand string `json:"start_command"`
	}
	if err := json.NewDecoder(stagingInfo).Decode(&info); err != nil {
		fatal(err, CodeFailedSetup, "parse start command")
	}
	return info.StartCommand
}

func fatal(err error, code int, action ...string) {
	message := "failed to " + strings.Join(action, " ")
	fmt.Fprintf(os.Stderr, "Error: %s: %s", message, err)
	os.Exit(code)
}
