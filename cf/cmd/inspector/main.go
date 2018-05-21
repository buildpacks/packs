package main

import (
	"flag"
	"fmt"
	"github.com/google/go-containerregistry/v1/remote"

	"github.com/sclevine/packs/cf/build"
	"github.com/sclevine/packs/cf/img"
	"github.com/sclevine/packs/cf/sys"
)

var (
	refName   string
	useDaemon bool
)

func init() {
	flag.BoolVar(&useDaemon, "daemon", sys.BoolEnv("PACK_USE_DAEMON"), "inspect image in docker daemon")
}

func main() {
	flag.Parse()
	refName = flag.Arg(0)
	if flag.NArg() != 1 || refName == "" {
		sys.Exit(sys.FailCode(sys.CodeInvalidArgs, "parse arguments"))
	}
	sys.Exit(inspect())
}

func inspect() error {
	store, err := img.NewRegistry(refName)
	if err != nil {
		return sys.FailErr(err, "access", refName)
	}
	image, err := store.Image()
	if err != nil {
		if rErr, ok := err.(*remote.Error); ok && len(rErr.Errors) > 0 {
			switch rErr.Errors[0].Code {
			case remote.UnauthorizedErrorCode, remote.ManifestUnknownErrorCode:
				return sys.FailCode(sys.CodeNotFound, "find", refName)
			}
		}
		return sys.FailErr(err, "get", refName)
	}
	config, err := image.ConfigFile()
	if err != nil {
		return sys.FailErr(err, "get config")
	}
	fmt.Println(config.Config.Labels[build.Label])
	return nil
}
