package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"bytes"

	cfapp "github.com/sclevine/packs/cf/app"
)

const (
	CodeFailedEnv = iota + 1
	CodeFailedSetup
	CodeFailedLaunch
)

const (
	homeDir         = "/home/vcap"
	appDir          = "/home/vcap/app"
	stagingInfoFile = "/home/vcap/staging_info.yml"
)

func main() {
	var droplet string
	flag.StringVar(&droplet, "droplet", os.Getenv("PACK_DROPLET_PATH"), "file containing compressed droplet")
	flag.Parse()

	command := strings.Join(flag.Args(), " ")

	if err := supplyApp(droplet, homeDir); err != nil {
		fatal(err, CodeFailedSetup, "supply app")
	}

	if err := os.Chdir(appDir); err != nil {
		fatal(err, CodeFailedSetup, "change directory to", appDir)
	}

	if command == "" {
		var err error
		command, err = readCommand(stagingInfoFile)
		if err != nil {
			fatal(err, CodeFailedSetup, "determine start command")
		}
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

	args := []string{"/lifecycle/launcher", appDir, command, ""}
	if err := syscall.Exec("/lifecycle/launcher", args, os.Environ()); err != nil {
		fatal(err, CodeFailedLaunch, "launch")
	}
}

func supplyApp(tgz, dst string) error {
	if _, err := os.Stat(tgz); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fail(err, "stat", tgz)
	}
	if err := runCmd("tar", "-C", dst, "-xzf", tgz); err != nil {
		return fail(err, "untar", tgz, "to", dst)
	}
	return nil
}

func readCommand(path string) (string, error) {
	stagingInfo, err := os.Open(path)
	if err != nil {
		return "", fail(err, "read start command")
	}
	var info struct {
		StartCommand string `json:"start_command"`
	}
	if err := json.NewDecoder(stagingInfo).Decode(&info); err != nil {
		return "", fail(err, "parse start command")
	}
	return info.StartCommand, nil
}

func runCmd(name string, arg ...string) error {
	buf := &bytes.Buffer{}
	cmd := exec.Command(name, arg...)
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %s\n%s", name, err, buf.String())
	}
}

func fail(err error, action ...string) error {
	message := "failed to " + strings.Join(action, " ")
	return fmt.Errorf("%s: %s", message, err)
}

func fatal(err error, code int, action ...string) {
	message := "failed to " + strings.Join(action, " ")
	fmt.Fprintf(os.Stderr, "Error: %s: %s", message, err)
	os.Exit(code)
}
