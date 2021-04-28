FROM alpine:3.13

ARG VERSION
ARG TARGETARCH

RUN set -ex; \
  FILENAME=cert-watcher-linux-$TARGETARCH; \
  wget -O /usr/local/bin/cert-watcher "https://github.com/RobertMe/cert-watcher/releases/download/$VERSION/$FILENAME"; \
  chmod +x /usr/local/bin/cert-watcher

CMD ['cert-watcher']

LABEL org.opencontainers.image.vendor="RobertMe" \
	org.opencontainers.image.url="https://github.com/RobertMe/cert-watcher" \
	org.opencontainers.image.title="cert-watcher" \
	org.opencontainers.image.description="cert-watcher allows you to, for example, extract Traefik certificates into a (running) container and optionally run a command in the container, or restart the container" \
	org.opencontainers.image.version="$VERSION" \
	org.opencontainers.image.documentation="https://github.com/RobertMe/cert-watcher"
