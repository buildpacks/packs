package sys

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ExitCode int

const (
	CodeFailed      ExitCode = 1
	CodeInvalidArgs          = iota + 3
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

func Fatal(err error, code ExitCode, action ...string) {
	var message string
	if len(action) > 0 {
		message = "failed to " + strings.Join(action, " ") + ": "
	}
	fmt.Fprintf(os.Stderr, "Error: %s%s\n", message, err)
	panic(code)
}

func Exit(code ExitCode, reason ...string) {
	if len(reason) > 0 {
		fmt.Fprintf(os.Stderr, "Exit: %s\n", strings.Join(reason, " "))
	}
	panic(code)
}

func Cleanup() {
	switch c := recover().(type) {
	case ExitCode:
		os.Exit(int(c))
	default:
		if c != nil {
			fmt.Fprintf(os.Stderr, "Crash: %s\n", c)
			os.Exit(int(CodeFailed))
		}
	}
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
