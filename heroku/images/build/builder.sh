#!/usr/bin/env bash

set -eu

# TODO allow buildpacks as args

BUILDPACKS_DIR=${1}
APP_DIR=${2}
SLUG_FILE=${3}
CACHE_FILE=${4}

detect="$(/packs/cytokine detect-buildpack --verbose ${APP_DIR} ${BUILDPACKS_DIR} 2>&1)"

buildpack="$(echo "${detect}" | grep -e '"https://.*"' -oh | sed -e 's/"//g')"

rm -rf /tmp/cache
mkdir -p /tmp/cache
if [ -f ${CACHE_FILE} ]; then
  tar xf ${CACHE_FILE} -C /tmp/cache/
fi

rm -rf /tmp/env
mkdir -p /tmp/env

/packs/cytokine run-buildpacks \
  --buildpack=${buildpack} \
  ${APP_DIR} \
  /tmp/cache \
  /tmp/env \
  ${BUILDPACKS_DIR}

# TODO run bin/release
/packs/cytokine release-buildpacks \
  --buildpack=${buildpack} \
  ${APP_DIR} \
  ${BUILDPACKS_DIR} \
  /tmp/release.yml

cat << EOF > ${APP_DIR}/staging_info.yml
{
  "detected_buildpack":"${buildpack}",
  "start_command":"${web_process}"
 }
EOF

/packs/cytokine make-slug /tmp/slug.tgz ${APP_DIR}

mkdir -p $(dirname ${SLUG_FILE})
mv /tmp/slug.tgz ${SLUG_FILE}

mkdir -p $(dirname ${CACHE_FILE})
tar czf ${CACHE_FILE} -C /tmp/cache/ .

# TODO create staging_info.yml

