ARG stack
FROM packs/${stack}

WORKDIR /workspace

EXPOSE 8080

COPY launcher /
COPY shell /

USER vcap

ENTRYPOINT ["/launcher", "-inputDroplet", "droplet.tgz"]
