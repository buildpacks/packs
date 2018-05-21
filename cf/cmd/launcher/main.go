package main

import (
	"encoding/json"
	"flag"
	"os"
	"strings"
	"syscall"

	cfapp "github.com/sclevine/packs/cf/app"
	"github.com/sclevine/packs/cf/build"
	"github.com/sclevine/packs/cf/sys"
)

const (
	homeDir         = "/home/vcap"
	appDir          = "/home/vcap/app"
	stagingInfoFile = "/home/vcap/staging_info.yml"
)

var (
	dropletPath  string
	metadataPath string
	startCommand string
)

func init() {
	flag.StringVar(&dropletPath, "droplet", os.Getenv("PACK_DROPLET_PATH"), "file containing compressed droplet")
	flag.StringVar(&metadataPath, "metadata", os.Getenv("PACK_DROPLET_METADATA_PATH"), "file containing droplet metadata")
}

func main() {
	flag.Parse()
	startCommand = strings.Join(flag.Args(), " ")
	sys.Exit(launch())
}

func launch() error {
	if err := supplyApp(dropletPath, homeDir); err != nil {
		return sys.FailErr(err, "supply app")
	}

	if err := os.Chdir(appDir); err != nil {
		return sys.FailErr(err, "change directory to", appDir)
	}

	var (
		command string
		err     error
	)
	switch {
	case startCommand != "":
		command = startCommand
	case metadataPath != "":
		command, err = readMetadataCommand(metadataPath)
	default:
		command, err = readDropletCommand(stagingInfoFile)
	}
	if err != nil {
		return sys.FailErr(err, "determine start command")
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

	args := []string{"/lifecycle/launcher", appDir, command, ""}
	if err := syscall.Exec("/lifecycle/launcher", args, os.Environ()); err != nil {
		return sys.FailErrCode(err, sys.CodeFailedLaunch, "launch")
	}
	return nil
}

func supplyApp(tgz, dst string) error {
	if _, err := os.Stat(tgz); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return sys.FailErr(err, "stat", tgz)
	}
	if _, err := sys.Run("tar", "-C", dst, "-xzf", tgz); err != nil {
		return sys.FailErr(err, "untar", tgz, "to", dst)
	}
	return nil
}

func readDropletCommand(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", sys.FailErr(err, "read droplet start command")
	}
	var info struct {
		StartCommand string `json:"start_command"`
	}
	if err := json.NewDecoder(f).Decode(&info); err != nil {
		return "", sys.FailErr(err, "parse start command")
	}
	return info.StartCommand, nil
}

func readMetadataCommand(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", sys.FailErr(err, "read metadata start command")
	}
	var metadata build.DropletMetadata
	if err := json.NewDecoder(f).Decode(&metadata); err != nil {
		return "", sys.FailErr(err, "parse start command")
	}
	return metadata.ProcessTypes["web"], nil
}
