# docker build . --file ./Dockerfile --build-arg version=latest -t dangnp/layerg-pluginbuilder:latest
# docker build . --file ./Dockerfile --build-arg version="v2.1.1-$(git rev-parse --short HEAD)" -t layerglabs/layerg-pluginbuilder:"2.1.1-$(git rev-parse --short HEAD)"

FROM arm64v8/golang:1.23-bookworm as builder

LABEL Layer Labs <support@layerg.xyz>

ARG version

LABEL version=$version
LABEL variant=layerg-pluginbuilder
LABEL description="A support container to build Go code for LayerG server's runtime."

ENV GOOS linux
ENV GOARCH amd64
ENV CGO_ENABLED 1

RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get install -y --no-install-recommends ca-certificates gcc libc6-dev git && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /builder

ENTRYPOINT ["go"]
