# syntax=docker/dockerfile:1
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY VERSION ./VERSION
ARG VERSION=2.0.0
ARG REVISION=development
RUN test "$(cat VERSION)" = "$VERSION" && CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.version=$VERSION -X main.revision=$REVISION" \
    -o /out/gsmsniffer ./cmd/server

FROM debian:bookworm-slim AS base
ARG VERSION=2.0.0
ARG REVISION=development
LABEL org.opencontainers.image.title="gsmsniffer" \
    org.opencontainers.image.description="Shielded, owner-operated GSM laboratory management" \
    org.opencontainers.image.source="https://github.com/addxemmm/gsmsniffer" \
    org.opencontainers.image.version=$VERSION \
    org.opencontainers.image.revision=$REVISION
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 gsmsniffer \
    && useradd --uid 10001 --gid 10001 --no-create-home --shell /usr/sbin/nologin gsmsniffer \
    && mkdir -p /var/lib/gsmsniffer && chown 10001:10001 /var/lib/gsmsniffer
COPY --from=build /out/gsmsniffer /usr/local/bin/gsmsniffer
ENV GSMSNIFFER_ADDR=:8080 GSMSNIFFER_MODE=demo \
    GSMSNIFFER_DATA_DIR=/var/lib/gsmsniffer GSMSNIFFER_MAX_DURATION_SECONDS=300
EXPOSE 8080
USER 10001:10001
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/usr/local/bin/gsmsniffer", "healthcheck"]
ENTRYPOINT ["/usr/local/bin/gsmsniffer"]

# Explicit opt-in target; not the default image. Packages include Python/GNU Radio.
FROM base AS shielded
USER root
RUN printf 'wireshark-common wireshark-common/install-setuid boolean false\n' | debconf-set-selections \
    && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends gr-gsm tshark \
    && rm -rf /var/lib/apt/lists/*
USER 10001:10001

# Keep the final/default target free of RF tools.
FROM base AS runtime
