FROM packs/cf2

ARG buildpacks

RUN mkdir /app /cache /out /home/vcap/tmp /tmp/buildpacks

RUN \
  for buildpack in ${buildpacks}; do \
    name=$(echo "${buildpack}" | cut -f1 -d=) && \
    url=$(echo "${buildpack}" | cut -f2- -d=) && \
    curl -fsSLo /tmp/buildpack.zip "$url" && \
    unzip /tmp/buildpack.zip -d "/tmp/buildpacks/$(echo -n "$name" | md5sum | awk '{ print $1 }')" && \
    rm /tmp/buildpack.zip; \
  done

COPY init /

USER vcap

ENV PATH=/usr/local/bin:/usr/bin:/bin LANG=en_US.UTF-8

ENTRYPOINT [
  "/init",
  "-buildDir", "/app",
  "-buildArtifactsCacheDir", "/cache",
  "-outputDroplet", "/out/droplet.tgz",
  "-outputMetadata", "/out/metadata.json"
  "-skipDetect",
  "-buildpackOrder"
]

