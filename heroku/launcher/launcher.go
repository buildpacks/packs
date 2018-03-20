package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	herokuapp "github.com/sclevine/packs/heroku/app"
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

	supplyApp(inputDroplet, "/")

	chownAll("heroku", "heroku", "/app")

	err := os.Chdir("/app")
	check(err, CodeFailedSetup, "change directory")

	if command == "" {
		command = readCommand("/staging_info.yml")
	}

	app, err := herokuapp.New()
	check(err, CodeFailedEnv, "build app env")
	for k, v := range app.Launch() {
		err := os.Setenv(k, v)
		check(err, CodeFailedEnv, "set app env")
	}

	args := []string{"/lifecycle/launcher", "/app", command, ""}
	err = syscall.Exec("/lifecycle/launcher", args, os.Environ())
	check(err, CodeFailedLaunch, "launch")
}

func supplyApp(tgz, dst string) {
	if _, err := os.Stat(tgz); os.IsNotExist(err) {
		return
	} else {
		check(err, CodeFailedSetup, "stat", tgz)
	}
	err := exec.Command("tar", "-C", dst, "-xzf", tgz).Run()
	check(err, CodeFailedSetup, "untar", tgz, "to", dst)
}

func readCommand(path string) string {
	stagingInfo, err := os.Open(path)
	check(err, CodeFailedSetup, "read start command")
	var info struct {
		StartCommand string `json:"start_command"`
	}
	err = json.NewDecoder(stagingInfo).Decode(&info)
	check(err, CodeFailedSetup, "parse start command")
	return info.StartCommand
}

func chownAll(user, group, path string) {
	err := exec.Command("chown", "-R", user+":"+group, path).Run()
	check(err, CodeFailedSetup, "chown", path, "to", user+":"+group)
}

func check(err error, code int, action ...string) {
	if err == nil {
		return
	}
	message := "failed to " + strings.Join(action, " ")
	fmt.Fprintf(os.Stderr, "Error: %s: %s", message, err)
	os.Exit(code)
}
