FROM packs/v3
ARG stack

COPY buildpacks /buildpacks

ENV PACK_BP_PATH /buildpacks
ENV PACK_BP_ORDER_PATH /buildpacks/order.toml
ENV PACK_BP_GROUP_PATH ./group.toml
ENV PACK_DETECT_INFO_PATH ./detect.toml
ENV PACK_STACK_NAME ${stack}

ENTRYPOINT ["/packs/detector"]
