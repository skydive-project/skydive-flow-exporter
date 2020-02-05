ARG  BASE=ubuntu:19.04
FROM $BASE
ARG  ARCH=amd64
RUN apt-get -y update \
    && apt-get -y install ca-certificates libpcap0.8 \
    && rm -rf /var/lib/apt/lists/*
COPY allinone.$ARCH /usr/bin/skydive-flow-exporter
COPY allinone/allinone.yml.default /etc/skydive-flow-exporter.yml
ENTRYPOINT ["/usr/bin/skydive-flow-exporter", "/etc/skydive-flow-exporter.yml"]
