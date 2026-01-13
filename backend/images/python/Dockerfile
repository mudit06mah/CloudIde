FROM python:3.12-slim-bullseye

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    git \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g 1500 dev && \
    useradd -m -u 1500 -g dev -s /bin/bash dev

RUN mkdir -p /workspace \
    && chown dev:dev /workspace

USER dev

ENV PIP_USER=true
ENV PATH="/home/dev/.local/bin:${PATH}"

WORKDIR /workspace
CMD [ "/bin/bash" ]