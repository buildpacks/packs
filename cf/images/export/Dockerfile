FROM gcr.io/cloud-builders/docker

ARG stack
ENV STACK "${stack}"

RUN mkdir /workspace
WORKDIR /workspace

RUN mkdir /packs
COPY Dockerfile.pack /packs/
COPY export /packs/

ENTRYPOINT ["/packs/export"]
