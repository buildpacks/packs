ARG stack
FROM packs/${stack}

ARG port=8080

WORKDIR /workspace

EXPOSE ${port}

RUN \
  apt-get update && \
  apt-get install -y sshpass && \
  rm -rf /var/lib/apt/lists/*

USER vcap