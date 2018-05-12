ARG stack
FROM packs/${stack}

ARG stack
ARG gcr_helper_version=1.4.1

RUN curl -fsSL https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v${gcr_helper_version}/docker-credential-gcr_linux_amd64-${gcr_helper_version}.tar.gz | \
    tar xz -O docker-credential-gcr > /usr/local/bin/docker-credential-gcr && \
    chmod +x /usr/local/bin/docker-credential-gcr

RUN mkdir -p /packs
COPY exporter /packs/

WORKDIR /workspace

ENV PACK_STACK_NAME packs/${stack}:run
ENV PACK_DROPLET_PATH ./droplet.tgz

ENTRYPOINT ["/packs/exporter"]
