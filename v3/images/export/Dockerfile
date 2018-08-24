ARG stack=packs/v3

FROM ${stack}

USER packs

ENV PACK_BP_GROUP_PATH ./group.toml
ENV PACK_METADATA_PATH ./metadata.toml

ENTRYPOINT ["/packs/exporter"]
