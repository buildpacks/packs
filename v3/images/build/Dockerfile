ARG go_version=1.10.1

FROM golang:${go_version} as deps

RUN go get github.com/sclevine/yj
RUN wget -qO bin/jq http://stedolan.github.io/jq/download/linux64/jq
RUN chmod +x bin/jq

FROM packs/v3:detect

COPY --from=deps /go/bin /usr/local/bin
ENV PACK_METADATA_PATH ./metadata.toml

ENTRYPOINT ["/packs/builder"]
