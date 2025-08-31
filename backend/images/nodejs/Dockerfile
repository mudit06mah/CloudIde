FROM node:18-bullseye-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    git \
    curl \
    ca-certificates \
    passwd \
 && rm -rf /var/lib/apt/lists/*

ARG UID=1000
ARG UNAME=dev

RUN if ! id -u ${UID} >/dev/null 2>&1; then \
      useradd -m -u ${UID} -s /bin/bash ${UNAME}; \
    fi \
 && mkdir -p /workspace \
 && chown -R ${UID}:${UID} /workspace

 
WORKDIR /workspace
USER ${UNAME}

CMD [ "/bin/bash" ]

EXPOSE 5173
