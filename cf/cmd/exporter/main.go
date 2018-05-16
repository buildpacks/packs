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
	"github.com/google/go-containerregistry/v1/mutate"
	"github.com/google/go-containerregistry/v1/remote"
	"github.com/google/go-containerregistry/v1/tarball"
	"github.com/google/image-rebase/pkg/rebase"
	"github.com/sclevine/packs/cf/sys"
)

func main() {
	defer sys.Cleanup()

	var (
		dropletTgz string
		stackRef   string
	)
	flag.StringVar(&dropletTgz, "droplet", os.Getenv("PACK_DROPLET_PATH"), "file containing compressed droplet")
	flag.StringVar(&stackRef, "stack", os.Getenv("PACK_STACK_NAME"), "base image for stack")
	flag.Parse()

	repo := flag.Arg(0)
	if flag.NArg() != 1 || repo == "" {
		sys.Exit(sys.CodeInvalidArgs, "invalid arguments")
	}
	registry := strings.ToLower(strings.SplitN(repo, "/", 2)[0])
	if err := configureCreds(registry, "gcr.io", "docker-credential-gcr", "configure-docker"); err != nil {
		sys.Fatal(err, sys.CodeFailed, "setup credentials")
	}

	var (
		oldStackDigest string
		newStackRef    string
		err            error
	)
	if dropletTgz != "" {
		newStackRef = stackRef
		oldStackDigest, err = appendDroplet(stackRef, repo, dropletTgz)
		if err != nil {
			sys.Fatal(err, sys.CodeFailedAppend, "append droplet")
		}
	}
	if err := rebaseDroplet(repo, oldStackDigest, newStackRef); err != nil {
		sys.Fatal(err, sys.CodeFailedRebase, "rebase droplet")
	}
}

func appendDroplet(stackImg, dstImg, dropletTgz string) (stackDigest string, err error) {
	tmpDir, err := ioutil.TempDir("", "packs.export")
	if err != nil {
		return "", sys.Fail(err, "create temp directory")
	}
	defer os.RemoveAll(tmpDir)

	dropletDir := filepath.Join(tmpDir, "home", "vcap")
	layerTGZ := filepath.Join(tmpDir, "layer.tgz")

	if err := os.MkdirAll(dropletDir, 0777); err != nil {
		return "", sys.Fail(err, "setup droplet directory")
	}
	if _, err := sys.Run("tar", "-C", dropletDir, "-xzf", dropletTgz); err != nil {
		return "", sys.Fail(err, "untar", dropletTgz, "to", dropletDir)
	}
	if _, err := sys.Run("chown", "-R", "vcap:vcap", dropletDir); err != nil {
		return "", sys.Fail(err, "recursively chown", dropletDir, "to", "vcap:vcap")
	}
	if _, err := sys.Run("tar", "-C", tmpDir, "-czf", layerTGZ, "home"); err != nil {
		return "", sys.Fail(err, "tar", tmpDir, "to", layerTGZ)
	}
	stackDigest, err = appendLayer(stackImg, dstImg, layerTGZ)
	if err != nil {
		return "", sys.Fail(err, "append", layerTGZ, "to", stackImg, "into", dstImg)
	}
	return stackDigest, nil
}

func appendLayer(src, dst, tar string) (srcDigest string, err error) {
	srcRef, err := name.ParseReference(src, name.WeakValidation)
	if err != nil {
		return "", err
	}

	srcAuth, err := authn.DefaultKeychain.Resolve(srcRef.Context().Registry)
	if err != nil {
		return "", err
	}

	srcImage, err := remote.Image(srcRef, srcAuth, http.DefaultTransport)
	if err != nil {
		return "", err
	}

	dstTag, err := name.NewTag(dst, name.WeakValidation)
	if err != nil {
		return "", err
	}

	layer, err := tarball.LayerFromFile(tar)
	if err != nil {
		return "", err
	}

	dstImage, err := mutate.AppendLayers(srcImage, layer)
	if err != nil {
		return "", err
	}

	opts := remote.WriteOptions{}
	if srcRef.Context().RegistryStr() == dstTag.Context().RegistryStr() {
		opts.MountPaths = append(opts.MountPaths, srcRef.Context())
	}

	dstAuth, err := authn.DefaultKeychain.Resolve(dstTag.Context().Registry)
	if err != nil {
		return "", err
	}

	srcDigestHash, err := srcImage.Digest()
	if err != nil {
		return "", err
	}
	srcDigest = srcRef.Context().Name() + "@" + srcDigestHash.String()
	return srcDigest, remote.Write(dstTag, dstImage, dstAuth, http.DefaultTransport, opts)
}

func rebaseDroplet(img, oldStackDigest, newStackRef string) error {
	rebaser := rebase.New(authn.DefaultKeychain, http.DefaultTransport)
	return rebaser.Rebase(img, oldStackDigest, newStackRef, img)
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
