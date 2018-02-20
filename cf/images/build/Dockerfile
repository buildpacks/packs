ARG stack
FROM packs/${stack}

ARG buildpacks

RUN \
  mkdir /var/lib/buildpacks && \
  echo "${buildpacks}" | jq -c '.[]' | while read row; do \
    name=$(echo "${row}" | jq -r '.name') && \
    uri=$(echo "${row}" | jq -r '.uri') && \
    curl -fsSLo /tmp/buildpack.zip "$uri" && \
    unzip -qq /tmp/buildpack.zip -d "/var/lib/buildpacks/$(echo -n "$name" | md5sum | awk '{ print $1 }')" && \
    rm /tmp/buildpack.zip; \
  done

COPY builder /

ENTRYPOINT [ \
  "/builder", \
  "-buildpacksDir", "/var/lib/buildpacks", \
  "-buildDir", "/app", \
  "-buildArtifactsCacheDir", "/cache", \
  "-outputDroplet", "/out/droplet.tgz", \
  "-outputBuildArtifactsCache", "/dev/null", \
  "-outputMetadata", "/out/result.json" \
]

