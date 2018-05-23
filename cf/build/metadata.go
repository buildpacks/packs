package build

import "code.cloudfoundry.org/buildpackapplifecycle"

const (
	BuildLabel = "sh.packs.build"
	BuildpackLabel = "sh.packs.buildpacks"
)

type Metadata struct {
	App        AppMetadata                               `json:"app"`
	Buildpacks []buildpackapplifecycle.BuildpackMetadata `json:"buildpacks"`
	Stack      StackMetadata                             `json:"stack"`
}

type DropletMetadata struct {
	buildpackapplifecycle.StagingResult
	PackMetadata PackMetadata `json:"pack_metadata"`
}

type PackMetadata struct {
	App AppMetadata `json:"app"`
}

type AppMetadata struct {
	Name string `json:"name"`
	SHA  string `json:"sha"`
}

type StackMetadata struct {
	Name string `json:"name"`
	SHA  string `json:"sha"`
}
