ARG stack

FROM golang:1.10-stretch as crane
ARG crane_commit=10ab64849078b8cb847aae0b5c05ee5d01842650

WORKDIR /go/src/github.com/google/go-containerregistry
RUN curl -fsSL "https://github.com/google/go-containerregistry/archive/${crane_commit}.tar.gz" | \
    tar xz --strip-components=1 && \
    go install github.com/google/go-containerregistry/cmd/crane

FROM packs/${stack}
ARG stack
ARG cred_helper_version=1.4.1

RUN curl -fsSL https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v${cred_helper_version}/docker-credential-gcr_linux_amd64-${cred_helper_version}.tar.gz | \
    tar xz -O docker-credential-gcr > /usr/local/bin/docker-credential-gcr && \
    chmod +x /usr/local/bin/docker-credential-gcr

COPY --from=crane /go/bin/crane /usr/local/bin/crane
ENV SRC_IMAGE packs/${stack}:run

WORKDIR /workspace

RUN mkdir /packs
COPY export /packs/

ENTRYPOINT ["/packs/export"]
