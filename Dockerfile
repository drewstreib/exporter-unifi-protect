FROM scratch

# goreleaser dockers_v2 lays binaries out per platform: <os>/<arch>/<binary>.
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/exporter-unifi-protect /usr/bin/exporter-unifi-protect

ENTRYPOINT [ "/usr/bin/exporter-unifi-protect" ]

CMD ["serve"]

