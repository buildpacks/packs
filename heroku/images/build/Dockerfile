ARG stack
FROM packs/${stack}

ARG buildpacks

WORKDIR /workspace

RUN mkdir -p /packs
RUN curl -o /packs/cytokine -L https://heroku-packs.s3.amazonaws.com/cytokine-a2a26fe7f9e1f05489e743fc55b863eb9079d94c
RUN chmod +x /packs/cytokine

RUN /packs/cytokine get-default-buildpacks \
  --language=ruby \
  --language=python \
  --language=java \
  --language=scala \
  --language=gradle \
  --language=clojure \
  --language=python \
  --language=go \
  --language=nodejs \
  /var/lib/buildpacks

COPY builder.sh /packs/

ENTRYPOINT [ \
  "/bin/bash", \
  "/packs/builder.sh", \
  "/var/lib/buildpacks", \
  "/tmp/app", \
  "/tmp/cache", \
  "/tmp/env", \
  "/out/slug.tgz", \
  "/cache/cache.tgz" \
]
