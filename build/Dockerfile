FROM packs/base

ARG buildpacks

USER root

RUN \
  mkdir -p /tmp/buildpacks && \
  for buildpack in ${buildpacks}; do \
    name=$(echo "${buildpack}" | cut -f1 -d=) && \
    url=$(echo "${buildpack}" | cut -f2- -d=) && \
    curl -fsSLo /tmp/buildpack.zip "$url" && \
    unzip /tmp/buildpack.zip -d "/tmp/buildpacks/$(echo -n "$name" | md5sum | awk '{ print $1 }')" && \
    rm /tmp/buildpack.zip; \
  done

USER vcap

CMD ["-buildpackOrder"]
