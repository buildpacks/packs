package img

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func Append(base v1.Image, tar string) (v1.Image, error) {
	layer, err := tarball.LayerFromFile(tar)
	if err != nil {
		return nil, err
	}
	image, err := mutate.AppendLayers(base, layer)
	if err != nil {
		return nil, err
	}
	return image, nil
}

type ImageFinder func(labels map[string]string) (v1.Image, error)

func Rebase(orig v1.Image, newBase v1.Image, oldBaseFinder ImageFinder) (v1.Image, error) {
	origConfig, err := orig.ConfigFile()
	if err != nil {
		return nil, err
	}
	oldBase, err := oldBaseFinder(origConfig.Config.Labels)
	if err != nil {
		return nil, err
	}
	image, err := mutate.Rebase(orig, oldBase, newBase, nil)
	if err != nil {
		return nil, err
	}
	return image, nil
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
