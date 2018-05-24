package img

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"

	"github.com/google/go-containerregistry/name"
	"github.com/google/go-containerregistry/v1"
	"github.com/google/go-containerregistry/v1/mutate"
	"github.com/google/go-containerregistry/v1/tarball"
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
		newBase.Ref().Context(),
		oldBase.Ref().Context(),
		orig.Ref().Context(),
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

type dockerConfig struct {
	CredHelpers map[string]string `json:"credHelpers"`
}

func SetupCredHelpers(refs ...string) error {
	dockerPath := filepath.Join(os.Getenv("HOME"), ".docker")
	configPath := filepath.Join(dockerPath, "config.json")
	if _, err := os.Stat(configPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	credHelpers := map[string]string{}
	for _, refStr := range refs {
		ref, err := name.ParseReference(refStr, name.WeakValidation)
		if err != nil {
			return err
		}
		registry := ref.Context().RegistryStr()
		for _, ch := range []struct {
			domain string
			helper string
		}{
			{"([.]|^)gcr[.]io$", "gcr"},
			{"[.]amazonaws[.]", "ecr-login"},
			{"([.]|^)azurecr[.]io$", "acr"},
		} {
			match, err := regexp.MatchString("(?i)"+ch.domain, registry)
			if err != nil || !match {
				continue
			}
			credHelpers[registry] = ch.helper
		}
	}
	if err := os.MkdirAll(dockerPath, 0777); err != nil {
		return err
	}
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(dockerConfig{
		CredHelpers: credHelpers,
	})
}
