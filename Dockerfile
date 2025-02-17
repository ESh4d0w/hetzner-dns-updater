FROM scratch
COPY hetzner-dns-updater /hetzner-dns-updater
COPY config.yaml /config.yaml
COPY --from=traefik:v2.11.20 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=traefik:v2.11.20 /usr/share/zoneinfo /usr/share/

VOLUME [ "/tmp" ]
ENTRYPOINT [ "/hetzner-dns-updater" ]
