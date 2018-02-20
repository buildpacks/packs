ARG stack
FROM packs/${stack}

ARG buildpacks

RUN \
  mkdir -p /var/lib/buildpacks && \
  for buildpack in ${buildpacks}; do \
    name=$(echo "${buildpack}" | cut -f1 -d:) && \
    url=$(echo "${buildpack}" | cut -f2- -d:) && \
    curl -fsSLo /tmp/buildpack.zip "$url" && \
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

