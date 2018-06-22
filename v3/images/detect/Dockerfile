FROM packs/v3
ARG stack

COPY buildpacks /buildpacks

ENV PACK_BP_PATH /buildpacks
ENV PACK_BP_ORDER_PATH /buildpacks/order.toml
ENV PACK_BP_GROUP_PATH ./group.toml
ENV PACK_DETECT_INFO_PATH ./detect.toml
ENV PACK_STACK_NAME ${stack}

RUN mkdir -p /launch/app
RUN chown -R packs:packs /launch

USER packs


ENTRYPOINT ["/packs/detector"]
