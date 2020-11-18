FROM golang:1.14 AS build
RUN apt-get -y update \
        && apt-get -y install libpcap-dev libpcap0.8 libpcap0.8-dev \
        && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /go/src/github.com/SODALITE-EU
WORKDIR /go/src/github.com/SODALITE-EU
RUN git clone https://github.com/SODALITE-EU/skydive-flow-exporter.git
WORKDIR /go/src/github.com/SODALITE-EU/skydive-flow-exporter/prom_sky_con
RUN go get -d
RUN make
FROM ubuntu:18.04
RUN apt-get -y update \
    && apt-get -y install openvswitch-common libpcap0.8 libxml2 libvirt0 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /go/bin/prom_sky_con /usr/bin/prom_sky_con
COPY --from=build /go/src/github.com/SODALITE-EU/skydive-flow-exporter/prom_sky_con/prom_sky_con.yml.default /etc/prom_sky_con.yml
EXPOSE 9100
ENTRYPOINT ["/usr/bin/prom_sky_con", "/etc/prom_sky_con.yml"]
