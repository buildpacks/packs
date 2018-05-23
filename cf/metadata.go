package cf

import (
	"code.cloudfoundry.org/buildpackapplifecycle"
	"github.com/sclevine/packs"
)

type DropletMetadata struct {
	buildpackapplifecycle.StagingResult
	PackMetadata packs.PackMetadata `json:"pack_metadata"`
}

func (d *DropletMetadata) Buildpacks() []packs.BuildpackMetadata {
	var out []packs.BuildpackMetadata
	for _, bp := range d.LifecycleMetadata.Buildpacks {
		out = append(out, packs.BuildpackMetadata(bp))
	}
	return out
}
