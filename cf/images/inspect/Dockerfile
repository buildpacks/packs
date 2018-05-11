ARG stack

FROM packs/${stack}:registry

RUN mkdir /packs
COPY inspector /packs/

WORKDIR /workspace

ENV PACK_IMAGE_METADATA_PATH ./image.json

ENTRYPOINT ["/packs/inspector"]
