ARG stack
FROM packs/${stack}

ARG buildpacks

WORKDIR /workspace

RUN mkdir -p /packs
RUN curl -o /packs/cytokine -L https://heroku-packs.s3.amazonaws.com/cytokine-afac8e5d6c604ed7c8156a3b92ceb79d337e1346
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
  "/workspace", \
  "/out/droplet.tgz", \
  "/cache/cache.tgz" \
]
