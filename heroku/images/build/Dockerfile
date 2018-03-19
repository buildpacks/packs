ARG stack
FROM packs/${stack}

ARG buildpacks

WORKDIR /workspace

RUN \
  mkdir /var/lib/buildpacks && \
  echo "${buildpacks}" > /var/lib/buildpacks/config.json && \
  echo "${buildpacks}" | jq -c '.[]' | while read row; do \
    name=$(echo "${row}" | jq -r '.name') && \
    uri=$(echo "${row}" | jq -r '.uri') && \
    key=$(echo -n "$name" | md5sum | awk '{ print $1 }') && \
    echo "Fetching: $name" && \
    curl -fsSLo /tmp/buildpack.tgz "$uri" && \
    mkdir -p /var/lib/buildpacks/$key && \
    echo "Unpacking: $name" && \
    tar xvf /tmp/buildpack.tgz -C /var/lib/buildpacks/$key > /dev/null 2>&1 && \
    rm /tmp/buildpack.tgz; \
  done

RUN mkdir /packs
COPY builder /packs/

ENTRYPOINT [ \
  "/packs/builder", \
  "-buildpacksDir", "/var/lib/buildpacks", \
  "-outputDroplet", "/out/droplet.tgz", \
  "-outputBuildArtifactsCache", "/cache/cache.tgz", \
  "-outputMetadata", "/out/result.json" \
]
