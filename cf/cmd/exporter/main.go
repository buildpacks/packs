package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/name"
	"github.com/google/go-containerregistry/v1"
	"github.com/sclevine/packs/cf/build"
	"github.com/sclevine/packs/cf/img"
	"github.com/sclevine/packs/cf/sys"
)

func main() {
	defer sys.Cleanup()

	var (
		dropletPath  string
		metadataPath string
		stackName    string
		local        bool
	)
	flag.StringVar(&dropletPath, "droplet", os.Getenv("PACK_DROPLET_PATH"), "file containing compressed droplet")
	flag.StringVar(&metadataPath, "metadata", os.Getenv("PACK_DROPLET_METADATA_PATH"), "file containing droplet metadata")
	flag.StringVar(&stackName, "stack", os.Getenv("PACK_STACK_NAME"), "base image for stack")
	flag.BoolVar(&local, "local", isTruthy(os.Getenv("PACK_IMAGE_LOCAL")), "export to docker daemon")
	flag.Parse()

	repoName := flag.Arg(0)
	if flag.NArg() != 1 || repoName == "" || stackName == "" {
		sys.Exit(sys.CodeInvalidArgs, "invalid arguments")
	}

	registry := strings.ToLower(strings.SplitN(repoName, "/", 2)[0])
	if err := configureCreds(registry, "gcr.io", "docker-credential-gcr", "configure-docker"); err != nil {
		sys.Fatal(err, sys.CodeFailed, "setup GCR credentials")
	}

	repoTag, err := name.NewTag(repoName, name.WeakValidation)
	if err != nil {
		sys.Fatal(err, sys.CodeInvalidArgs, "parse repository", repoName)
	}
	var repoStore img.Store
	if local {
		repoStore = img.NewDaemon(repoTag)
	} else {
		repoStore, err = img.NewRegistry(repoTag)
		if err != nil {
			sys.Fatal(err, sys.CodeFailed, "access", repoName)
		}
	}

	stackRef, err := name.ParseReference(stackName, name.WeakValidation)
	if err != nil {
		sys.Fatal(err, sys.CodeInvalidArgs, "parse stack", stackName)
	}
	stackStore, err := img.NewRegistry(stackRef)
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "access", stackName)
	}

	var (
		repoImage       v1.Image
		dropletMetadata *build.DropletMetadata
		buildMetadata   build.Metadata
	)
	if dropletPath != "" {
		if metadataPath != "" {
			metadataFile, err := os.Open(metadataPath)
			if err != nil {
				sys.Fatal(err, sys.CodeFailed, "failed to open", metadataPath)
			}
			defer metadataFile.Close()
			dropletMetadata = &build.DropletMetadata{}
			if err := json.NewDecoder(metadataFile).Decode(&dropletMetadata); err != nil {
				sys.Fatal(err, sys.CodeFailed, "failed to decode metadata")
			}
		}
		layer, err := dropletToLayer(dropletPath)
		if err != nil {
			sys.Fatal(err, sys.CodeFailed, "transform", dropletPath, "into layer")
		}
		defer os.Remove(layer)
		repoImage, err = img.Append(stackStore, layer)
		if err != nil {
			sys.Fatal(err, sys.CodeFailed, "append droplet to", stackName)
		}
		repoStore.Source(stackRef.Context())
	} else {
		var sources []name.Repository
		repoImage, sources, err = img.Rebase(repoStore, stackStore, func(labels map[string]string) (img.Store, error) {
			if err := json.Unmarshal([]byte(labels[build.Label]), &buildMetadata); err != nil {
				sys.Fatal(err, sys.CodeFailed, "get build metadata for", repoName)
			}
			digestName := buildMetadata.Stack.Name + "@" + buildMetadata.Stack.SHA
			oldStackDigest, err := name.NewDigest(digestName, name.WeakValidation)
			if err != nil {
				return nil, err
			}
			return img.NewRegistry(oldStackDigest)
		})
		if err != nil {
			sys.Fatal(err, sys.CodeFailed, "rebase", repoName, "on", stackName)
		}
		repoStore.Source(sources...)
	}
	stackDigest, err := stackStore.Digest()
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "get digest for", stackName)
	}
	buildMetadata.Stack.Name = stackRef.Context().String()
	buildMetadata.Stack.SHA = stackDigest.String()
	if dropletMetadata != nil {
		buildMetadata.App = dropletMetadata.PackMetadata.App
		buildMetadata.Buildpacks = dropletMetadata.Buildpacks
	}
	buildJSON, err := json.Marshal(buildMetadata)
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "get encode metadata for", repoName)
	}
	repoImage, err = img.Label(repoImage, build.Label, string(buildJSON))
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "label", repoName)
	}
	if err := repoStore.Write(repoImage); err != nil {
		sys.Fatal(err, sys.CodeFailedUpdate, "write", repoName)
	}
}

func dropletToLayer(dropletPath string) (layer string, err error) {
	tmpDir, err := ioutil.TempDir("", "pack.export.layer")
	if err != nil {
		return "", sys.Fail(err, "create temp directory")
	}
	defer os.RemoveAll(tmpDir)

	layer = tmpDir + ".tgz"
	dropletRoot := filepath.Join(tmpDir, "home", "vcap")

	if err := os.MkdirAll(dropletRoot, 0777); err != nil {
		return "", sys.Fail(err, "setup droplet directory")
	}
	if _, err := sys.Run("tar", "-C", dropletRoot, "-xzf", dropletPath); err != nil {
		return "", sys.Fail(err, "untar", dropletPath, "to", dropletRoot)
	}
	if _, err := sys.Run("chown", "-R", "vcap:vcap", dropletRoot); err != nil {
		return "", sys.Fail(err, "recursively chown", dropletRoot, "to", "vcap:vcap")
	}
	if _, err := sys.Run("tar", "-C", tmpDir, "-czf", layer, "home"); err != nil {
		defer os.Remove(layer)
		return "", sys.Fail(err, "tar", tmpDir, "to", layer)
	}
	return layer, nil
}

func configureCreds(registry, domain, command string, args ...string) error {
	if registry == domain || strings.HasSuffix(registry, "."+domain) {
		if _, err := sys.Run(command, args...); err != nil {
			return sys.Fail(err, "configure", domain, "credentials")
		}
		fmt.Printf("Configured credentials for: %s\n", domain)
	}
	return nil
}

func isTruthy(s string) bool {
	return s == "true" || s == "1"
}
