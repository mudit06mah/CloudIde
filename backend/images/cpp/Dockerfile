FROM gcc:12-bullseye

RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    curl \
    cmake \
    gdb \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 -s /bin/bash dev

RUN mkdir -p /workspace && chown dev:dev /workspace

USER dev

WORKDIR /workspace
CMD [ "/bin/bash" ]