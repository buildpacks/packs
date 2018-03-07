ARG stack
FROM packs/${stack}:export

ARG cred_helper_version=1.4.1

RUN curl -L https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v${cred_helper_version}/docker-credential-gcr_linux_amd64-${cred_helper_version}.tar.gz | \
    tar xz -O docker-credential-gcr > /usr/local/bin/docker-credential-gcr && \
    chmod +x /usr/local/bin/docker-credential-gcr

RUN docker-credential-gcr configure-docker
