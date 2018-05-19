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
	"github.com/google/go-containerregistry/v1/mutate"
	"github.com/google/go-containerregistry/v1/tarball"

	"github.com/sclevine/packs/cf/build"
	"github.com/sclevine/packs/cf/sys"
	"github.com/sclevine/packs/cf/img"
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
	flag.BoolVar(&local, "local", isTruthy(os.Getenv("PACK_IMAGE_LOCAL")), "export to local docker daemon")
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
		repoStore, err = img.NewRepo(repoTag)
		if err != nil {
			sys.Fatal(err, sys.CodeFailed, "access", repoName)
		}
	}

	stackRef, err := name.ParseReference(stackName, name.WeakValidation)
	if err != nil {
		sys.Fatal(err, sys.CodeInvalidArgs, "parse stack", stackName)
	}
	stackStore, err := img.NewRepo(stackRef)
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "access", stackName)
	}
	stackImage, err := stackStore.Image()
	if err != nil {
		sys.Fatal(err, sys.CodeNotFound, "get image", stackName)
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
		repoImage, err = appendLayer(stackImage, layer)
		if err != nil {
			sys.Fatal(err, sys.CodeFailed, "append droplet to", stackName, "for", repoName)
		}
		repoStore.Source(stackRef)
	} else {
		origImage, err := repoStore.Image()
		if err != nil {
			sys.Fatal(err, sys.CodeNotFound, "get image", repoName)
		}
		origConfig, err := origImage.ConfigFile()
		if err != nil {
			sys.Fatal(err, sys.CodeFailed, "get config file for", repoName)
		}
		if err := json.Unmarshal([]byte(origConfig.Config.Labels[build.Label]), &buildMetadata); err != nil {
			sys.Fatal(err, sys.CodeFailed, "get build metadata for", repoName)
		}
		var oldBaseRef name.Reference
		repoImage, oldBaseRef, err = rebaseLayer(origImage, stackImage, buildMetadata.Stack)
		if err != nil {
			sys.Fatal(err, sys.CodeFailed, "rebase", repoName, "on", stackName)
		}
		repoStore.Source(oldBaseRef, stackRef)
	}
	repoConfigFile, err := repoImage.ConfigFile()
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "get config file for", repoName)
	}
	repoConfig := *repoConfigFile.Config.DeepCopy()
	if repoConfig.Labels == nil {
		repoConfig.Labels = map[string]string{}
	}
	stackDigest, err := stackImage.Digest()
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
	repoConfig.Labels[build.Label] = string(buildJSON)

	repoImage, err = mutate.Config(repoImage, repoConfig)
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "set config file for", repoName)
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

func appendLayer(origImage v1.Image, tar string) (image v1.Image, err error) {
	layer, err := tarball.LayerFromFile(tar)
	if err != nil {
		return nil, err
	}
	return mutate.AppendLayers(origImage, layer)
}

func rebaseLayer(origImage, newStackImage v1.Image, oldStack build.StackMetadata) (image v1.Image, oldStackRef name.Reference, err error) {
	oldStackDigest, err := name.NewDigest(oldStack.Name+"@"+oldStack.SHA, name.WeakValidation)
	if err != nil {
		return nil, nil, err
	}
	oldStackStore, err := img.NewRepo(oldStackDigest)
	if err != nil {
		return nil, nil, err
	}
	oldStackImage, err := oldStackStore.Image()
	if err != nil {
		return nil, nil, err
	}
	image, err = mutate.Rebase(origImage, oldStackImage, newStackImage, nil)
	if err != nil {
		return nil, nil, err
	}
	return image, oldStackDigest, nil
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
