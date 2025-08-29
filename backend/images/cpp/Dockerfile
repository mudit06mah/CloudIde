FROM debian:bookworm_slim
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    git \
    curl \
    ca-certificates && rm -rf /var/lib/apt/lists/*
ARG UID=1000
RUN useradd -m -u ${UID} dev || true
RUN mkdir -p /workspace && chown -R {UID}:{UID} /workspace
WORKDIR /workspace
USER {UID}
ENTRYPOINT [ "bin/bash" ]
