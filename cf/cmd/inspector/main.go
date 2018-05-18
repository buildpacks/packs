package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-containerregistry/authn"
	"github.com/google/go-containerregistry/name"
	"github.com/google/go-containerregistry/v1/remote"

	"github.com/sclevine/packs/cf/build"
	"github.com/sclevine/packs/cf/sys"
)

func main() {
	defer sys.Cleanup()

	var metadataPath string
	flag.StringVar(&metadataPath, "metadata", os.Getenv("PACK_IMAGE_METADATA_PATH"), "path for image metadata output")
	flag.Parse()

	input := flag.Arg(0)
	if flag.NArg() != 1 || input == "" {
		sys.Exit(sys.CodeInvalidArgs, "invalid arguments")
	}

	ref, err := name.ParseReference(input, name.WeakValidation)
	if err != nil {
		sys.Fatal(err, sys.CodeInvalidArgs, "parse reference")
	}
	auth, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
	if err != nil {
		sys.Fatal(err, sys.CodeFailedInspect, "authenticate registry")
	}
	img, err := remote.Image(ref, auth, http.DefaultTransport)
	if err != nil {
		sys.Fatal(err, sys.CodeFailedInspect, "locate image")
	}
	digest, err := img.Digest()
	if err != nil {
		if rErr, ok := err.(*remote.Error); ok && len(rErr.Errors) > 0 {
			switch rErr.Errors[0].Code {
			case remote.UnauthorizedErrorCode, remote.ManifestUnknownErrorCode:
				fmt.Fprintf(os.Stderr, "Not found.")
				sys.Exit(sys.CodeNotFound)
			}
		}
		sys.Fatal(err, sys.CodeFailedInspect, "determine digest")
	}
	config, err := img.ConfigFile()
	if err != nil {
		sys.Fatal(err, sys.CodeFailedInspect, "retrieve manifest")
	}
	metadataFile, err := os.Create(metadataPath)
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "create metadata file", metadataPath)
	}
	defer metadataFile.Close()

	if err := json.NewEncoder(metadataFile).Encode(config.Config.Labels[build.Label]); err != nil {
		defer os.RemoveAll(metadataPath)
		sys.Fatal(err, sys.CodeFailed, "write metadata to", metadataPath)
	}
	fmt.Println(ref.Context().Name() + "@" + digest.String())
}
