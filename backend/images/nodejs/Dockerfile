FROM node:18-bullseye-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    git \
    curl \
    ca-certificates \
    passwd \
 && rm -rf /var/lib/apt/lists/*

ARG UID=1500
ARG UNAME=dev

RUN useradd -m -u ${UID} -s /bin/bash ${UNAME} \
 && mkdir -p /workspace \
 && chown -R ${UID}:${UID} /workspace

WORKDIR /workspace

# Let user pick user at runtime; default to root
CMD ["/bin/bash"]

EXPOSE 5173
