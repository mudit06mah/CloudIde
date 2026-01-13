FROM node:20-bullseye-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create user 'dev' with ID 1500
RUN groupadd -g 1500 dev && \
    useradd -m -u 1500 -g dev -s /bin/bash dev

RUN mkdir -p /workspace && chown dev:dev /workspace

USER dev

ENV NPM_CONFIG_PREFIX=/home/dev/.npm-global
ENV PATH=$PATH:/home/dev/.npm-global/bin

WORKDIR /workspace
CMD [ "/bin/bash" ]