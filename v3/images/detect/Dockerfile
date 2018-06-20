ARG stack
FROM packs/${stack}

COPY buildpacks /buildpacks
COPY order.toml /order.toml

ENV PACK_BP_PATH /buildpacks/
ENV PACK_BP_ORDER_PATH /order.toml
ENV PACK_BP_GROUP_PATH ./group.toml
ENV PACK_DETECT_INFO_PATH ./detect.toml

ENTRYPOINT ["/packs/detector"]
