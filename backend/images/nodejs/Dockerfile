FROM node:20-bullseye-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# The 'node' user (UID 1000) already exists in this image.
# We just configure NPM to work for this user.

USER node

# Setup global npm directory for the 'node' user
ENV NPM_CONFIG_PREFIX=/home/node/.npm-global
ENV PATH=$PATH:/home/node/.npm-global/bin

WORKDIR /workspace
CMD [ "/bin/bash" ]