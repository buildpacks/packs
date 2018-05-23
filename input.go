package packs

import (
	"flag"
	"os"
)

func InputDropletPath(path *string) {
	flag.StringVar(path, "droplet", os.Getenv("PACK_DROPLET_PATH"), "file containing compressed droplet")
}

func InputMetadataPath(path *string) {
	flag.StringVar(path, "metadata", os.Getenv("PACK_DROPLET_METADATA_PATH"), "file containing droplet metadata")
}

func InputStackName(name *string) {
	flag.StringVar(name, "stack", os.Getenv("PACK_STACK_NAME"), "image repository containing stack")
}

func InputUseDaemon(use *bool) {
	flag.BoolVar(use, "daemon", BoolEnv("PACK_USE_DAEMON"), "export to docker daemon")
}
