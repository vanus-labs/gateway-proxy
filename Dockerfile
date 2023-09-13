FROM --platform=$BUILDPLATFORM golang:1.18 as builder
WORKDIR /workspace

ARG TARGETOS
ARG TARGETARCH

COPY . .
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make build

FROM --platform=$TARGETPLATFORM ubuntu:22.04

RUN apt-get update && apt-get install -y \
        ca-certificates \
        tzdata \
        && update-ca-certificates \
        && rm -rf /var/lib/apt/lists/*

WORKDIR /vanus
ARG git_commit
COPY --from=builder /workspace/bin/proxy /vanus/bin/proxy

ENV GIT_HASH=${git_commit}

CMD ["bin/proxy"]
