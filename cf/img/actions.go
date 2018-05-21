package img

import (
	"github.com/google/go-containerregistry/name"
	"github.com/google/go-containerregistry/v1"
	"github.com/google/go-containerregistry/v1/mutate"
	"github.com/google/go-containerregistry/v1/tarball"
	"strings"
	"os/exec"
	"github.com/sclevine/packs/cf/sys"
)

func Append(s Store, tar string) (v1.Image, []name.Repository, error) {
	baseImage, err := s.Image()
	if err != nil {
		return nil, nil, err
	}
	layer, err := tarball.LayerFromFile(tar)
	if err != nil {
		return nil, nil, err
	}
	image, err := mutate.AppendLayers(baseImage, layer)
	if err != nil {
		return nil, nil, err
	}
	return image, []name.Repository{s.Ref().Context()}, nil
}

type StoreFinder func(labels map[string]string) (Store, error)

func Rebase(orig Store, newBase Store, oldBaseFinder StoreFinder) (v1.Image, []name.Repository, error) {
	origImage, err := orig.Image()
	if err != nil {
		return nil, nil, err
	}
	origConfig, err := origImage.ConfigFile()
	if err != nil {
		return nil, nil, err
	}
	oldBase, err := oldBaseFinder(origConfig.Config.Labels)
	if err != nil {
		return nil, nil, err
	}
	oldBaseImage, err := oldBase.Image()
	if err != nil {
		return nil, nil, err
	}
	newBaseImage, err := newBase.Image()
	if err != nil {
		return nil, nil, err
	}
	image, err := mutate.Rebase(origImage, oldBaseImage, newBaseImage, nil)
	if err != nil {
		return nil, nil, err
	}
	return image, []name.Repository{
		orig.Ref().Context(),
		newBase.Ref().Context(),
		oldBase.Ref().Context(),
	}, nil
}

func Label(image v1.Image, k, v string) (v1.Image, error) {
	configFile, err := image.ConfigFile()
	if err != nil {
		return nil, err
	}
	config := *configFile.Config.DeepCopy()
	if config.Labels == nil {
		config.Labels = map[string]string{}
	}
	config.Labels[k] = v
	return mutate.Config(image, config)
}

func RunInDomain(ref, domain, cmd string, args ...string) (bool, error) {
	r, err := name.ParseReference(ref, name.WeakValidation)
	if err != nil {
		return false, err
	}
	registry := strings.ToLower(r.Context().RegistryStr())
	domain = strings.ToLower(domain)
	if registry != domain && !strings.HasSuffix(registry, "."+domain) {
		return false, nil
	}
	if _, err := exec.LookPath(cmd); err != nil {
		return false, nil
	}
	if _, err := sys.Run(cmd, args...); err != nil {
		return false, err
	}
	return true, nil
}