package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"flag"

	"github.com/google/go-containerregistry/authn"
	"github.com/google/go-containerregistry/name"
	"github.com/google/go-containerregistry/v1/remote"
	"encoding/json"
)

const (
	CodeNotFound = iota + 1
	CodeFailedParse
	CodeFailedInspect
	CodeInvalidArgs
)

type Metadata struct {
	Ref string            `json:"ref"`
	Rev map[string]string `json:"rev"`
}

func main() {
	var metadataPath string
	flag.StringVar(&metadataPath, "metadata", os.Getenv("PACK_IMAGE_METADATA_PATH"), "path for image metadata output")
	flag.Parse()

	input := flag.Arg(0)
	if flag.NArg() != 1 || input == "" {
		fatal(nil, CodeInvalidArgs, "parse arguments")
	}

	ref, err := name.ParseReference(input, name.WeakValidation)
	if err != nil {
		fatal(err, CodeFailedParse, "parse reference")
	}
	auth, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
	if err != nil {
		fatal(err, CodeFailedInspect, "authenticate registry")
	}
	img, err := remote.Image(ref, auth, http.DefaultTransport)
	if err != nil {
		fatal(err, CodeFailedInspect, "locate image")
	}
	digest, err := img.Digest()
	if err != nil {
		if rErr, ok := err.(*remote.Error); ok && len(rErr.Errors) > 0 {
			switch rErr.Errors[0].Code {
			case remote.UnauthorizedErrorCode, remote.ManifestUnknownErrorCode:
				fmt.Fprintf(os.Stderr, "Not found.")
				os.Exit(CodeNotFound)
			}
		}
		fatal(err, CodeFailedInspect, "determine digest")
	}
	manifest, err := img.Manifest()
	if err != nil {
		fatal(err, CodeFailedInspect, "retrieve manifest")
	}
	metadata := Metadata{
		Ref: ref.Context().Name() + "@" + digest.String(),
		Rev: manifest.Annotations,
	}
	metadataFile, err := os.Create(metadataPath)
	if err != nil {
		fatal(err, CodeFailedInspect, "create file", metadataPath)
	}
	defer metadataFile.Close()

	if err := json.NewEncoder(metadataFile).Encode(metadata); err != nil {
		defer os.RemoveAll(metadataPath)
		fatal(err, CodeFailedInspect, "write metadata to", metadataPath)
	}
	fmt.Println(metadata.Ref)
}

func fatal(err error, code int, action ...string) {
	message := "failed to " + strings.Join(action, " ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s: %s\n", message, err)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	}
	os.Exit(code)
}
