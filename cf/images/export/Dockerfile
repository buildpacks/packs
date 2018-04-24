# set the stack arg outside of the build stage so that it can be referenced by the FROM instructions
# https://docs.docker.com/engine/reference/builder/#understand-how-arg-and-from-interact
ARG stack

FROM golang:1.10-stretch as cli-builder
WORKDIR /go/src/github.com/google
ARG D8S_COMMIT=21ca7dca3eca6f287b663eab23af6d858547c2e5
RUN curl -L -o source.tgz \
      "https://github.com/google/go-containerregistry/archive/$D8S_COMMIT.tar.gz" && \
    tar xzf source.tgz && \
    mv "go-containerregistry-$D8S_COMMIT" go-containerregistry && \
    rm source.tgz && \
    go install github.com/google/go-containerregistry/cmd/d8s


FROM packs/${stack}
COPY --from=cli-builder /go/bin/d8s /usr/local/bin/d8s
# redeclare the stack arg since this is now within a build stage
# https://docs.docker.com/engine/reference/builder/#understand-how-arg-and-from-interact
ARG stack
ENV SRC_IMAGE packs/${stack}:run

RUN mkdir /workspace
WORKDIR /workspace

RUN mkdir /packs
COPY export /packs/
ENTRYPOINT ["/packs/export"]
