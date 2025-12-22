FROM scratch

ARG VERSION
ARG TARGETARCH

ADD --chmod=700 https://github.com/RobertMe/cert-watcher/releases/download/$VERSION/cert-watcher-linux-$TARGETARCH /cert-watcher

CMD ["/cert-watcher"]

LABEL org.opencontainers.image.vendor="RobertMe" \
	org.opencontainers.image.url="https://github.com/RobertMe/cert-watcher" \
	org.opencontainers.image.title="cert-watcher" \
	org.opencontainers.image.description="cert-watcher allows you to, for example, extract Traefik certificates into a (running) container and optionally run a command in the container, or restart the container" \
	org.opencontainers.image.version="$VERSION" \
	org.opencontainers.image.documentation="https://github.com/RobertMe/cert-watcher"
