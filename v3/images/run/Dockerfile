FROM packs/v3

EXPOSE 8080

RUN apt-get update && \
  apt-get install -y wget && \
  rm -rf /var/lib/apt/lists/*

HEALTHCHECK --interval=30s --timeout=1s --start-period=60s --retries=1 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/ || exit 1

USER packs

ENV PACK_PROCESS_TYPE web
ENV PACK_METADATA_PATH /launch/config/metadata.toml

ENTRYPOINT ["/packs/launcher"]
