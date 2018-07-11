#!/usr/bin/env bash
set -euo pipefail
set -x

IMAGE=myorg/myapp ## Change to your own

sudo rm -rf /tmp/packsv3/{platform,launch,workspace}
mkdir -p /tmp/packsv3/{platform,cache,launch,workspace}
cp -r ~/workspace/nodejs-buildpack/fixtures/node_version_range /tmp/packsv3/launch/app
# rm /tmp/packsv3/launch/app/node_modules

docker run \
  -v /tmp/packsv3/launch/app:/launch/app \
  -v /tmp/packsv3/workspace:/workspace \
  packs/v3:detect
touch /tmp/packsv3/workspace/detect.toml

# ## Temp
# ./v3/bin/build packs/cflinuxfs2
# docker run -it -w /launch/app -v /tmp/packsv3/cache:/cache -v /tmp/packsv3/launch:/launch -v /tmp/packsv3/launch/app:/launch/app --entrypoint bash packs/v3:build
# exit 1

 ## TODO: platform is provided, but not used. Here for future input. TODO: Should it be?
docker run \
  -v /tmp/packsv3/platform:/platform \
  -v /tmp/packsv3/cache:/cache \
  -v /tmp/packsv3/launch:/launch \
  -v /tmp/packsv3/workspace:/workspace \
  packs/v3:build

docker run \
  --user 0 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /tmp/packsv3/launch:/launch \
  -v /tmp/packsv3/workspace:/workspace \
  packs/v3:export \
  -daemon \
  -stack packs/v3 \
  "$IMAGE"

docker run -e PORT=8080 -p 8080:8080 "$IMAGE"
