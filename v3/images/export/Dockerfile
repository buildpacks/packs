FROM packs/v3

RUN apt-get update && \
  apt-get install -y ca-certificates && \
  rm -rf /var/lib/apt/lists/*

USER packs

ENV PACK_BP_GROUP_PATH ./group.toml
ENV PACK_METADATA_PATH ./metadata.toml

ENTRYPOINT ["/packs/exporter"]
