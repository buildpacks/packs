FROM gcr.io/cloud-builders/docker

ARG stack
ENV STACK "${stack}"

RUN mkdir /workspace
WORKDIR /workspace

COPY Dockerfile.pack /
COPY export /

ENTRYPOINT ["/export"]
