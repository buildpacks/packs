package build

import "code.cloudfoundry.org/buildpackapplifecycle"

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
	Name    string `json:"name"`
	Version string `json:"version"`
}

type StackMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
