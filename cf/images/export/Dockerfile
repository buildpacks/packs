ARG stack

FROM packs/${stack}:registry
ARG stack

RUN mkdir /packs
COPY exporter /packs/

WORKDIR /workspace

ENV PACK_STACK_NAME packs/${stack}:run
ENV PACK_DROPLET_PATH ./droplet.tgz

ENTRYPOINT ["/packs/exporter"]
