FROM golang:1.21-bullseye

RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 -s /bin/bash dev

RUN mkdir -p /workspace && chown dev:dev /workspace

USER dev

ENV GOPATH=/home/dev/go
ENV PATH=$PATH:$GOPATH/bin

WORKDIR /workspace
CMD [ "/bin/bash" ]