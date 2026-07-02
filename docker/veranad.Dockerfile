# syntax=docker/dockerfile:1.5
ARG GO_VERSION=1.26.4
ARG BASE_IMAGE=ubuntu:22.04

FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /app

ARG USE_PREBUILT=false
ARG PREBUILT_BINARY=binaries/veranad-linux-amd64
ARG LDFLAGS=""

# Separate layer so source edits don't re-trigger the module download.
COPY go.mod go.sum ./
RUN if [ "$USE_PREBUILT" != "true" ]; then go mod download; fi

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    if [ "$USE_PREBUILT" = "true" ] && [ -f "$PREBUILT_BINARY" ]; then \
      cp "$PREBUILT_BINARY" /tmp/veranad; \
    else \
      CGO_ENABLED=0 go build -ldflags="$LDFLAGS" -o /tmp/veranad ./cmd/veranad; \
    fi

FROM ${BASE_IMAGE}

ARG INCLUDE_GO=false

RUN apt-get update && apt-get install -y \
    bash \
    ca-certificates \
    curl \
    jq \
    python3 \
    s3cmd \
    tini \
    wget \
    $(if [ "$INCLUDE_GO" = "true" ]; then echo golang; fi) \
  && rm -rf /var/lib/apt/lists/*

COPY --chmod=0755 --from=builder /tmp/veranad /usr/local/bin/veranad

EXPOSE 26656 26657 1317 9090

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["veranad"]
