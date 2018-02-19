FROM packs/${stack}

ARG stack
ARG buildpacks

RUN mkdir /app /cache /out /home/vcap/tmp

RUN \
  mkdir -p /tmp/buildpacks && \
  for buildpack in ${buildpacks}; do \
    name=$(echo "${buildpack}" | cut -f1 -d=) && \
    url=$(echo "${buildpack}" | cut -f2- -d=) && \
    curl -fsSLo /tmp/buildpack.zip "$url" && \
    unzip /tmp/buildpack.zip -d "/tmp/buildpacks/$(echo -n "$name" | md5sum | awk '{ print $1 }')" && \
    rm /tmp/buildpack.zip; \
  done

COPY builder /

ENTRYPOINT [
  "/builder",
  "-buildDir", "/app",
  "-buildArtifactsCacheDir", "/cache",
  "-outputDroplet", "/out/droplet.tgz",
  "-outputBuildArtifactsCache", "/dev/null",
  "-outputMetadata", "/out/metadata.json"
  "-skipDetect",
  "-buildpackOrder"
]

