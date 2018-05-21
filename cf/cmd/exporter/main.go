package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/name"
	"github.com/google/go-containerregistry/v1"

	"code.cloudfoundry.org/buildpackapplifecycle"
	"github.com/sclevine/packs/cf/build"
	"github.com/sclevine/packs/cf/img"
	"github.com/sclevine/packs/cf/sys"
)

var (
	dropletPath  string
	metadataPath string
	repoName     string
	stackName    string
	useDaemon    bool
)

func init() {
	flag.StringVar(&dropletPath, "droplet", os.Getenv("PACK_DROPLET_PATH"), "file containing compressed droplet")
	flag.StringVar(&metadataPath, "metadata", os.Getenv("PACK_DROPLET_METADATA_PATH"), "file containing droplet metadata")
	flag.StringVar(&stackName, "stack", os.Getenv("PACK_STACK_NAME"), "base image for stack")
	flag.BoolVar(&useDaemon, "daemon", sys.BoolEnv("PACK_USE_DAEMON"), "export to docker daemon")
}

func main() {
	flag.Parse()
	repoName = flag.Arg(0)
	if flag.NArg() != 1 || repoName == "" || stackName == "" || (metadataPath != "" && dropletPath == "") {
		sys.Exit(sys.FailCode(sys.CodeInvalidArgs, "parse arguments"))
	}
	sys.Exit(export())
}

func export() error {
	if ran, err := img.RunInDomain(repoName, "gcr.io", "docker-credential-gcr", "configure-docker"); err != nil {
		return sys.FailErr(err, "setup GCR credentials")
	} else if ran {
		fmt.Println("Configured GCR credentials.")
	}

	newRepoStore := img.NewRegistry
	if useDaemon {
		newRepoStore = img.NewDaemon
	}
	repoStore, err := newRepoStore(repoName)
	if err != nil {
		return sys.FailErr(err, "access", repoName)
	}

	stackStore, err := img.NewRegistry(stackName)
	if err != nil {
		return sys.FailErr(err, "access", stackName)
	}

	var (
		repoImage v1.Image
		sources   []name.Repository
		metadata  build.Metadata
	)
	if dropletPath != "" {
		if metadataPath != "" {
			var err error
			metadata.App, metadata.Buildpacks, err = readDropletMetadata(metadataPath)
			if err != nil {
				return sys.FailErr(err, "get droplet metadata")
			}
		}
		layer, err := dropletToLayer(dropletPath)
		if err != nil {
			return sys.FailErr(err, "transform", dropletPath, "into layer")
		}
		defer os.Remove(layer)
		repoImage, sources, err = img.Append(stackStore, layer)
		if err != nil {
			return sys.FailErr(err, "append droplet to", stackName)
		}
	} else {
		repoImage, sources, err = img.Rebase(repoStore, stackStore, func(labels map[string]string) (img.Store, error) {
			if err := json.Unmarshal([]byte(labels[build.Label]), &metadata); err != nil {
				return nil, sys.FailErr(err, "get build metadata for", repoName)
			}
			digestName := metadata.Stack.Name + "@" + metadata.Stack.SHA
			return img.NewRegistry(digestName)
		})
		if err != nil {
			return sys.FailErr(err, "rebase", repoName, "on", stackName)
		}
	}
	stackDigest, err := stackStore.Digest()
	if err != nil {
		return sys.FailErr(err, "get digest for", stackName)
	}
	metadata.Stack.Name = stackStore.Ref().Context().String()
	metadata.Stack.SHA = stackDigest.String()
	buildJSON, err := json.Marshal(metadata)
	if err != nil {
		return sys.FailErr(err, "get encode metadata for", repoName)
	}
	repoImage, err = img.Label(repoImage, build.Label, string(buildJSON))
	if err != nil {
		return sys.FailErr(err, "label", repoName)
	}
	if err := repoStore.Write(repoImage, sources...); err != nil {
		return sys.FailErrCode(err, sys.CodeFailedUpdate, "write", repoName)
	}
	return nil
}

func readDropletMetadata(path string) (build.AppMetadata, []buildpackapplifecycle.BuildpackMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return build.AppMetadata{}, nil, sys.FailErr(err, "failed to open", path)
	}
	defer f.Close()
	metadata := build.DropletMetadata{}
	if err := json.NewDecoder(f).Decode(&metadata); err != nil {
		return build.AppMetadata{}, nil, sys.FailErr(err, "failed to decode", path)
	}
	return metadata.PackMetadata.App, metadata.Buildpacks, nil
}

func dropletToLayer(dropletPath string) (layer string, err error) {
	tmpDir, err := ioutil.TempDir("", "pack.export.layer")
	if err != nil {
		return "", sys.FailErr(err, "create temp directory")
	}
	defer os.RemoveAll(tmpDir)

	layer = tmpDir + ".tgz"
	dropletRoot := filepath.Join(tmpDir, "home", "vcap")

	if err := os.MkdirAll(dropletRoot, 0777); err != nil {
		return "", sys.FailErr(err, "setup droplet directory")
	}
	if _, err := sys.Run("tar", "-C", dropletRoot, "-xzf", dropletPath); err != nil {
		return "", sys.FailErr(err, "untar", dropletPath, "to", dropletRoot)
	}
	if _, err := sys.Run("chown", "-R", "vcap:vcap", dropletRoot); err != nil {
		return "", sys.FailErr(err, "recursively chown", dropletRoot, "to", "vcap:vcap")
	}
	if _, err := sys.Run("tar", "-C", tmpDir, "-czf", layer, "home"); err != nil {
		defer os.Remove(layer)
		return "", sys.FailErr(err, "tar", tmpDir, "to", layer)
	}
	return layer, nil
}
