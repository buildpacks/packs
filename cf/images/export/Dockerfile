ARG stack

FROM golang:1.10-stretch as d8s
ARG d8s_commit=21ca7dca3eca6f287b663eab23af6d858547c2e5

WORKDIR /go/src/github.com/google/go-containerregistry
RUN curl -fsSL "https://github.com/google/go-containerregistry/archive/${d8s_commit}.tar.gz" | \
    tar xz --strip-components=1 && \
    go install github.com/google/go-containerregistry/cmd/d8s

FROM packs/${stack}
ARG stack
ARG cred_helper_version=1.4.1

RUN curl -fsSL https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v${cred_helper_version}/docker-credential-gcr_linux_amd64-${cred_helper_version}.tar.gz | \
    tar xz -O docker-credential-gcr > /usr/local/bin/docker-credential-gcr && \
    chmod +x /usr/local/bin/docker-credential-gcr

COPY --from=d8s /go/bin/d8s /usr/local/bin/d8s
ENV SRC_IMAGE packs/${stack}:run

WORKDIR /workspace

RUN mkdir /packs
COPY export /packs/

ENTRYPOINT ["/packs/export"]
