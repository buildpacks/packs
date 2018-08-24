FROM packs/v3:detect

USER root

RUN mkdir -p /cache /launch/app /launch/config /platform/env
RUN chown -R packs:packs /launch /cache

USER packs

ENV PACK_METADATA_PATH /launch/config/metadata.toml

ENTRYPOINT ["/packs/builder"]
