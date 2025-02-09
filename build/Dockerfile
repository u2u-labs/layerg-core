# docker build "$PWD" --build-arg commit="$(git rev-parse --short HEAD)" --build-arg version=latest -t dangnp/layerg:latest
# docker build "$PWD" --build-arg commit="$(git rev-parse --short HEAD)" --build-arg version="$(git rev-parse --short HEAD)" -t layerglabs/layerg:"$(git rev-parse --short HEAD)"

FROM golang:1.23-bookworm AS builder

ARG commit
ARG version

ENV GOOS linux
ENV GOARCH amd64
ENV CGO_ENABLED 1

RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get install -y --no-install-recommends ca-certificates gcc libc6-dev git && \
    git config --global advice.detachedHead false && \
    git clone --quiet --no-checkout https://github.com/u2u-labs/layerg-core /go/build/layerg

WORKDIR /go/build/layerg
COPY . .

RUN git checkout --quiet "$commit" && \
    go mod tidy && \
    go mod vendor && \
    go build -o /go/build-out/layerg -trimpath -mod=vendor -gcflags "-trimpath $PWD" -asmflags "-trimpath $PWD" -ldflags "-s -w -X main.version=$version -X main.commitID=$commit"

FROM debian:bookworm-slim

LABEL LayerG Labs <support@layerg.xyz>

ARG version

LABEL version=$version
LABEL variant=layerg
LABEL description="Distributed server for social and realtime games and apps."

RUN mkdir -p /layerg/data/modules && \
    apt-get update && \
    apt-get -y upgrade && \
    apt-get install -y --no-install-recommends ca-certificates tzdata iproute2 tini && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /layerg/
COPY --from=builder "/go/build-out/layerg" /layerg/
EXPOSE 7349 7350 7351

ENTRYPOINT ["tini", "--", "/layerg/layerg"]

HEALTHCHECK --interval=30s --timeout=10s \
  CMD /layerg/layerg healthcheck || exit 1
