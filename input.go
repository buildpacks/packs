package packs

import (
	"flag"
	"os"
)

const (
	EnvAppDir    = "PACK_APP_DIR"
	EnvAppZip    = "PACK_APP_ZIP"
	EnvAppName   = "PACK_APP_NAME"
	EnvAppURI    = "PACK_APP_URI"
	EnvAppDisk   = "PACK_APP_DISK"
	EnvAppMemory = "PACK_APP_MEM"
	EnvAppFds    = "PACK_APP_FDS"

	EnvDropletPath  = "PACK_DROPLET_PATH"
	EnvMetadataPath = "PACK_DROPLET_METADATA_PATH"
	EnvStackName    = "PACK_STACK_NAME"
	EnvUseDaemon    = "PACK_USE_DAEMON"
)

func InputDropletPath(path *string) {
	flag.StringVar(path, "droplet", os.Getenv(EnvDropletPath), "file containing compressed droplet")
}

func InputMetadataPath(path *string) {
	flag.StringVar(path, "metadata", os.Getenv(EnvMetadataPath), "file containing droplet metadata")
}

func InputStackName(name *string) {
	flag.StringVar(name, "stack", os.Getenv(EnvStackName), "image repository containing stack")
}

func InputUseDaemon(use *bool) {
	flag.BoolVar(use, "daemon", BoolEnv(EnvUseDaemon), "export to docker daemon")
}
