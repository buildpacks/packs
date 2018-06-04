package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"

	"github.com/sclevine/packs"
	"github.com/sclevine/packs/cf"
	"github.com/sclevine/packs/img"
)

var (
	dropletPath  string
	metadataPath string
	repoName     string
	stackName    string
	useDaemon    bool
)

func init() {
	packs.InputDropletPath(&dropletPath)
	packs.InputMetadataPath(&metadataPath)
	packs.InputStackName(&stackName)
	packs.InputUseDaemon(&useDaemon)
}

func main() {
	flag.Parse()
	repoName = flag.Arg(0)
	if flag.NArg() != 1 || repoName == "" || stackName == "" || (metadataPath != "" && dropletPath == "") {
		packs.Exit(packs.FailCode(packs.CodeInvalidArgs, "parse arguments"))
	}
	packs.Exit(export())
}

func export() error {
	if err := img.SetupCredHelpers(repoName, stackName); err != nil {
		return packs.FailErr(err, "setup credential helper")
	}

	newRepoStore := img.NewRegistry
	if useDaemon {
		newRepoStore = img.NewDaemon
	}
	repoStore, err := newRepoStore(repoName)
	if err != nil {
		return packs.FailErr(err, "access", repoName)
	}

	stackStore, err := img.NewRegistry(stackName)
	if err != nil {
		return packs.FailErr(err, "access", stackName)
	}

	var (
		repoImage v1.Image
		sources   []name.Repository
		metadata  packs.BuildMetadata
	)
	if dropletPath != "" {
		if metadataPath != "" {
			var err error
			metadata.App, metadata.Buildpacks, err = readDropletMetadata(metadataPath)
			if err != nil {
				return packs.FailErr(err, "get droplet metadata")
			}
		}
		layer, err := dropletToLayer(dropletPath)
		if err != nil {
			return packs.FailErr(err, "transform", dropletPath, "into layer")
		}
		defer os.Remove(layer)
		repoImage, sources, err = img.Append(stackStore, layer)
		if err != nil {
			return packs.FailErr(err, "append droplet to", stackName)
		}
	} else {
		repoImage, sources, err = img.Rebase(repoStore, stackStore, func(labels map[string]string) (img.Store, error) {
			if err := json.Unmarshal([]byte(labels[packs.BuildLabel]), &metadata); err != nil {
				return nil, packs.FailErr(err, "get build metadata for", repoName)
			}
			digestName := metadata.Stack.Name + "@" + metadata.Stack.SHA
			return img.NewRegistry(digestName)
		})
		if err != nil {
			return packs.FailErr(err, "rebase", repoName, "on", stackName)
		}
	}
	stackDigest, err := stackStore.Digest()
	if err != nil {
		return packs.FailErr(err, "get digest for", stackName)
	}
	metadata.Stack.Name = stackStore.Ref().Context().String()
	metadata.Stack.SHA = stackDigest.String()
	buildJSON, err := json.Marshal(metadata)
	if err != nil {
		return packs.FailErr(err, "get encode metadata for", repoName)
	}
	repoImage, err = img.Label(repoImage, packs.BuildLabel, string(buildJSON))
	if err != nil {
		return packs.FailErr(err, "label", repoName)
	}
	if err := repoStore.Write(repoImage, sources...); err != nil {
		return packs.FailErrCode(err, packs.CodeFailedUpdate, "write", repoName)
	}
	return nil
}

func readDropletMetadata(path string) (packs.AppMetadata, []packs.BuildpackMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return packs.AppMetadata{}, nil, packs.FailErr(err, "failed to open", path)
	}
	defer f.Close()
	var metadata cf.DropletMetadata
	if err := json.NewDecoder(f).Decode(&metadata); err != nil {
		return packs.AppMetadata{}, nil, packs.FailErr(err, "failed to decode", path)
	}
	return metadata.PackMetadata.App, metadata.Buildpacks(), nil
}

func dropletToLayer(dropletPath string) (layer string, err error) {
	tmpDir, err := ioutil.TempDir("", "pack.export.layer")
	if err != nil {
		return "", packs.FailErr(err, "create temp directory")
	}
	defer os.RemoveAll(tmpDir)

	layer = tmpDir + ".tgz"
	dropletRoot := filepath.Join(tmpDir, "home", "vcap")

	if err := os.MkdirAll(dropletRoot, 0777); err != nil {
		return "", packs.FailErr(err, "setup droplet directory")
	}
	if _, err := packs.Run("tar", "-C", dropletRoot, "-xzf", dropletPath); err != nil {
		return "", packs.FailErr(err, "untar", dropletPath, "to", dropletRoot)
	}
	if _, err := packs.Run("chown", "-R", "vcap:vcap", dropletRoot); err != nil {
		return "", packs.FailErr(err, "recursively chown", dropletRoot, "to", "vcap:vcap")
	}
	if _, err := packs.Run("tar", "-C", tmpDir, "-czf", layer, "home"); err != nil {
		defer os.Remove(layer)
		return "", packs.FailErr(err, "tar", tmpDir, "to", layer)
	}
	return layer, nil
}
