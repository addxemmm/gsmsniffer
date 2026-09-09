# syntax=docker/dockerfile:1
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY VERSION ./VERSION
ARG VERSION=2.1
ARG REVISION=development
RUN test "$(cat VERSION)" = "$VERSION" && CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.version=$VERSION -X main.revision=$REVISION" \
    -o /out/gsmsniffer ./cmd/server

# Match the operator's working LTE/GSM UHD 4.1 stack. Jammy does not package
# gr-gsm, so build its pinned GNU Radio 3.10 source against Jammy libraries.
FROM ubuntu:22.04 AS radio-build
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    ca-certificates curl xz-utils build-essential cmake pkg-config gnuradio-dev \
    libosmocore-dev gr-osmosdr python3-dev python3-matplotlib python3-numpy \
    python3-scipy python3-six libcppunit-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /src
RUN curl -fsSL --retry 3 https://deb.debian.org/debian/pool/main/g/gr-gsm/gr-gsm_1.0.0~20220727.orig.tar.xz -o gr-gsm.tar.xz \
    && echo '968f262dd7090b1d5951a5c743296c8ba063cc71127667dacb18aa4bcdb22153  gr-gsm.tar.xz' | sha256sum -c - \
    && tar -xf gr-gsm.tar.xz \
    && cmake -S gr-gsm-1.0.0~20220727 -B build \
       -DCMAKE_INSTALL_PREFIX=/usr -DCMAKE_BUILD_TYPE=Release \
       -DLIB_SUFFIX=/$(dpkg-architecture -qDEB_HOST_MULTIARCH) \
       -DPYTHON_EXECUTABLE=/usr/bin/python3 -DGR_PYTHON_DIR=/usr/lib/python3/dist-packages \
       -DENABLE_PYTHON=ON -DENABLE_GRC=ON -DENABLE_GRCC=ON \
       -DENABLE_GSM_LIVEMON_HEADLESS=ON -DENABLE_DOXYGEN=OFF \
    && cmake --build build --parallel 2 \
    && DESTDIR=/stage cmake --install build \
    && mkdir -p /stage/usr/share/doc/gr-gsm \
    && cp gr-gsm.tar.xz /stage/usr/share/doc/gr-gsm/source.tar.xz \
    && cp gr-gsm-1.0.0~20220727/COPYING /stage/usr/share/doc/gr-gsm/COPYING

FROM ubuntu:22.04 AS base
ARG VERSION=2.1
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
ENV GSMSNIFFER_ADDR=:18083 GSMSNIFFER_API_ADDR=:8083 GSMSNIFFER_MODE=demo \
    GSMSNIFFER_DATA_DIR=/var/lib/gsmsniffer GSMSNIFFER_MAX_DURATION_SECONDS=300
EXPOSE 18083 8083
USER 10001:10001
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/usr/local/bin/gsmsniffer", "healthcheck"]
ENTRYPOINT ["/usr/local/bin/gsmsniffer"]

# Dependencies are included in the single published image; RF remains opt-in.
FROM base AS shielded
USER root
RUN printf 'wireshark-common wireshark-common/install-setuid boolean false\n' | debconf-set-selections \
    && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
       gnuradio gr-osmosdr uhd-host tshark libosmocore18 libosmogsm17 libosmocoding0 libosmocodec0 \
       python3-matplotlib python3-numpy python3-scipy python3-six \
    && rm -rf /var/lib/apt/lists/*
COPY --from=radio-build /stage/usr/ /usr/
RUN ldconfig && uhd_config_info --version | grep 'UHD 4.1.' \
    && python3 -c 'from gnuradio import gsm; import osmosdr' \
    && grgsm_scanner --help >/dev/null \
    && grgsm_livemon_headless --help >/dev/null
USER 10001:10001

# One image for both modes; the default remains unprivileged demo.
FROM shielded AS runtime
