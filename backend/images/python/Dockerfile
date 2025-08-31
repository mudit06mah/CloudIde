FROM python:3.12-slim-bullseye

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    git \
    curl \
    ca-certificates && rm -rf /var/lib/apt/lists/*

ARG UID=1000
ARG UNAME=dev

RUN useradd -m -u ${UID} -s /bin/bash ${UNAME} || true \
 && mkdir -p /workspace \
 && chown -R ${UID}:${UID} /workspace

WORKDIR /workspace
USER ${UNAME}

ENTRYPOINT [ "/bin/bash" ]