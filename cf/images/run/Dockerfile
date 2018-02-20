ARG stack
FROM packs/${stack}

EXPOSE 8080

COPY launcher /

USER vcap

ENTRYPOINT ["/launcher", "-inputDroplet", "/in/droplet.tgz"]
