package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/authn"
	"github.com/google/go-containerregistry/name"
	"github.com/google/go-containerregistry/v1"
	"github.com/google/go-containerregistry/v1/mutate"
	"github.com/google/go-containerregistry/v1/remote"
	"github.com/google/go-containerregistry/v1/tarball"
	"github.com/sclevine/packs/cf/sys"
	"github.com/sclevine/packs/cf/build"
	"encoding/json"
)

func main() {
	defer sys.Cleanup()

	var (
		dropletPath string
		metadataPath string
		stackName   string
	)
	flag.StringVar(&dropletPath, "droplet", os.Getenv("PACK_DROPLET_PATH"), "file containing compressed droplet")
	flag.StringVar(&metadataPath, "metadata", os.Getenv("PACK_DROPLET_METADATA_PATH"), "file containing droplet metadata")
	flag.StringVar(&stackName, "stack", os.Getenv("PACK_STACK_NAME"), "base image for stack")
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
		sys.Fatal(err, sys.CodeInvalidArgs, "parse repository reference:", repoName)
	}
	repoAuth, err := authn.DefaultKeychain.Resolve(repoTag.Context().Registry)
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "authenticate with repository registry")
	}

	stackRef, err := name.ParseReference(stackName, name.WeakValidation)
	if err != nil {
		sys.Fatal(err, sys.CodeInvalidArgs, "parse stack reference:", stackName)
	}
	stackAuth, err := authn.DefaultKeychain.Resolve(stackRef.Context().Registry)
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "authenticate with stack registry")
	}
	stackImage, err := remote.Image(stackRef, stackAuth, http.DefaultTransport)
	if err != nil {
		sys.Fatal(err, sys.CodeNotFound, "locate stack image:", stackName)
	}

	var (
		repoImage       v1.Image
		repoMounts      []name.Repository
		dropletMetadata *build.DropletMetadata
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
		if stackRef.Context().RegistryStr() == repoTag.Context().RegistryStr() {
			repoMounts = []name.Repository{stackRef.Context()}
		}
	} else {
		origImage, err := remote.Image(repoTag, repoAuth, http.DefaultTransport)
		if err != nil {
			sys.Fatal(err, sys.CodeNotFound, "locate repository image:", repoName)
		}
		var oldBaseRef name.Reference
		repoImage, oldBaseRef, err = rebaseLayer(origImage, stackImage)
		if err != nil {
			sys.Fatal(err, sys.CodeFailed, "rebase", repoName, "on", stackName)
		}
		// TODO: consider filtering on repoTag.Context().RegistryStr() and excluding repoTag.Context()
		repoMounts = []name.Repository{repoTag.Context(), oldBaseRef.Context(), stackRef.Context()}
	}
	repoConfig, err := repoImage.ConfigFile()
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "get config for", repoName)
	}
	stackDigest, err := stackImage.Digest()
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "get digest for", stackName)
	}
	var buildMetadata build.Metadata
	if err := json.Unmarshal([]byte(repoConfig.Config.Labels[build.Label]), &buildMetadata); err != nil {
		sys.Fatal(err, sys.CodeFailed, "get build metadata for", repoName)
	}
	buildMetadata.Stack.Name = stackRef.Context().String()
	buildMetadata.Stack.Version = stackDigest.String()
	if dropletMetadata != nil {
		buildMetadata.App = dropletMetadata.PackMetadata.App
		buildMetadata.Buildpacks = dropletMetadata.Buildpacks
	}
	buildJSON, err := json.Marshal(buildMetadata)
	if err != nil {
		sys.Fatal(err, sys.CodeFailed, "get encode metadata for", repoName)
	}
	repoConfig.Config.Labels[build.Label] = string(buildJSON)
	if err := remote.Write(repoTag, repoImage, repoAuth, http.DefaultTransport, remote.WriteOptions{
		MountPaths: repoMounts,
	}); err != nil {
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

func rebaseLayer(origImage, newBaseImage v1.Image) (image v1.Image, oldBaseRef name.Reference, err error) {
	origConfig, err := origImage.ConfigFile()
	if err != nil {
		return nil, nil, err
	}
	oldBaseName := origConfig.Config.Labels["pack.stack.name"] + "@" + origConfig.Config.Labels["pack.stack.version"]
	oldBaseDigest, err := name.NewDigest(oldBaseName, name.WeakValidation)
	if err != nil {
		return nil, nil, err
	}
	oldBaseAuth, err := authn.DefaultKeychain.Resolve(oldBaseDigest.Context().Registry)
	if err != nil {
		return nil, nil, err
	}
	oldBaseImage, err := remote.Image(oldBaseDigest, oldBaseAuth, http.DefaultTransport)
	if err != nil {
		return nil, nil, err
	}
	image, err = mutate.Rebase(origImage, oldBaseImage, newBaseImage, nil)
	if err != nil {
		return nil, nil, err
	}
	return image, oldBaseDigest, nil
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
