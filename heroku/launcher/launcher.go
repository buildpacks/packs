package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"gopkg.in/yaml.v2"

	herokuapp "github.com/sclevine/packs/heroku/app"
)

const (
	CodeFailedEnv = iota + 1
	CodeFailedSetup
	CodeFailedLaunch
)

type ErrProcfileNoProcess string

func (e ErrProcfileNoProcess) Error() string {
	return fmt.Sprintf("%s", e)
}

type ErrNoCommandFound string

func (e ErrNoCommandFound) Error() string {
	return fmt.Sprintf("%s", e)
}

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
		command, err = readCommand()
		check(err, CodeFailedSetup, "please add a Procfile with a web process")
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

func readCommand() (string, error) {
	if command, err := parseProcfile("/app/Procfile"); err == nil {
		return command, nil
	} else if command, err = parseReleaseYml("/app/release.yml"); err == nil {
		return command, nil
	}
	return "", ErrNoCommandFound("No command found, please specify one in your Procfile.")
}

func parseProcfile(path string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		buf, err := ioutil.ReadFile(path)
		procfile := string(buf)
		check(err, CodeFailedSetup, "parse Procfile")

		processes := make(map[string]string)

		for _, line := range strings.Split(procfile, "\n") {
			array := strings.SplitAfterN(line, ":", 2)
			if len(array) == 2 {
				processes[array[0]] = array[1]
			}
		}

		if process, ok := processes["web:"]; ok {
			return process, nil
		}
	}

	return "", ErrProcfileNoProcess("No web process in Procfile.")
}

func parseReleaseYml(path string) (string, error) {
	releaseYml, err := ioutil.ReadFile(path)
	check(err, CodeFailedSetup, "read start command")
	var info struct {
		Addons              []string          `yaml:"addons"`
		DefaultProcessTypes map[string]string `yaml:"default_process_types"`
	}
	err = yaml.Unmarshal(releaseYml, &info)
	if err == nil {
		return info.DefaultProcessTypes["web"], nil
	} else {
		return "", err
	}
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
