package packs

import (
	"flag"
	"os"
)

const (
	EnvAppDir = "PACK_APP_DIR"
	EnvAppZip = "PACK_APP_ZIP"

	EnvAppName = "PACK_APP_NAME"
	EnvAppURI  = "PACK_APP_URI"

	EnvAppDisk   = "PACK_APP_DISK"
	EnvAppMemory = "PACK_APP_MEM"
	EnvAppFds    = "PACK_APP_FDS"

	EnvDropletPath  = "PACK_DROPLET_PATH"
	EnvSlugPath     = "PACK_SLUG_PATH"
	EnvMetadataPath = "PACK_METADATA_PATH"

	EnvStackName  = "PACK_STACK_NAME"
	EnvUseDaemon  = "PACK_USE_DAEMON"
	EnvUseHelpers = "PACK_USE_HELPERS"
)

func InputDropletPath(path *string) {
	flag.StringVar(path, "droplet", os.Getenv(EnvDropletPath), "file containing droplet")
}

func InputSlugPath(path *string) {
	flag.StringVar(path, "slug", os.Getenv(EnvSlugPath), "file containing slug")
}

func InputMetadataPath(path *string) {
	flag.StringVar(path, "metadata", os.Getenv(EnvMetadataPath), "file containing artifact metadata")
}

func InputStackName(image *string) {
	flag.StringVar(image, "stack", os.Getenv(EnvStackName), "image repository containing stack image")
}

func InputUseDaemon(use *bool) {
	flag.BoolVar(use, "daemon", boolEnv(EnvUseDaemon), "export to docker daemon")
}

func InputUseHelpers(use *bool) {
	flag.BoolVar(use, "helpers", boolEnv(EnvUseHelpers), "use credential helpers")
}

func boolEnv(k string) bool {
	v := os.Getenv(k)
	return v == "true" || v == "1"
}
