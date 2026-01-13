FROM gcc:12-bullseye

RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    curl \
    cmake \
    gdb \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g 1500 dev && \
    useradd -m -u 1500 -g dev -s /bin/bash dev

RUN mkdir -p /workspace && chown dev:dev /workspace

USER dev

WORKDIR /workspace
CMD [ "/bin/bash" ]