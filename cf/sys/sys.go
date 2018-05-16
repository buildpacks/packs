package sys

import (
	"strings"
	"fmt"
	"os"
	"bytes"
	"os/exec"
)

const (
	CodeFailed = 1

	CodeInvalidArgs = iota + 3
	CodeInvalidEnv
	CodeNotFound
	CodeFailedBuild
	CodeFailedLaunch
	CodeFailedInspect
	CodeFailedAppend
	CodeFailedRebase
)

func Fail(err error, action ...string) error {
	message := "failed to " + strings.Join(action, " ")
	return fmt.Errorf("%s: %s", message, err)
}

func Fatal(err error, code int, action ...string) {
	var message string
	if len(action) > 0 {
		message = "failed to " + strings.Join(action, " ") + ": "
	}
	fmt.Fprintf(os.Stderr, "Error: %s%s\n", message, err)
	os.Exit(code)
}

func Exit(code int, reason ...string) {
	if len(reason) > 0 {
		fmt.Fprintf(os.Stderr, "Exit: %s\n", strings.Join(reason, " "))
	}
	os.Exit(code)
}

func Run(name string, arg ...string) (string, error) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd := exec.Command(name, arg...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s failed: %s\n%s", name, err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}