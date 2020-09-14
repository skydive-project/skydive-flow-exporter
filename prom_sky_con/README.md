# Overview

The Prometheus-Skydive connector prepares Skydive flow information into a format 
suitable to be consumed by Prometheus.
The connector is built upon the core Skydive Flow Exporter and extracts basic 
information about each captured flow to provide to Prometheus.

# Deploy

## Deploy Skydive

Follow insutructions on http://skydive.network/documentation/getting-started to download and start up skydive.

You will need at least one Skydive analyzer to which the agents will connect.

```
skydive analyzer [--conf etc/skydive.yml]
```

You will need a Skydive agent to run on each of your hosts.
Be sure to set the `analyzers` setting in the yml file to point to the network address of the analyzer.

```
skydive agent [--conf etc/skydive.yml]
```

To run Skydive analyzer as a container, run the following:
```
docker run --privileged --pid=host --net=host -p 8082:8082 skydive/skydive analyzer
```
To run Skydive agent as a container, you can run the following:
```
docker run --privileged --pid=host --net=host -p 8081:8081  -v /var/run/libvirt/libvirt-sock-ro:/var/run/libvirt/libvirt-sock-ro -v /var/run/docker.sock:/var/run/docker.sock -e SKYDIVE_AGENT_TOPOLOGY_PROBES="lldp libvirt k8s" -e SKYDIVE_ANALYZERS=localhost:8082 skydive/skydive agent
```
Set the proper network address for SKYDIVE_ANALYZERS.

## Create Capture

Setup capture on every interface:

```
skydive client capture create \
	--gremlin "G.V().Has('Type', 'device', 'Name', 'eth0')" \
	--type pcap \
	-c etc/skydive.yml.default
```

To perform this using a container:

```
docker run --net=host skydive/skydive client capture create \
	--type pcap \
	--gremlin "G.V().Has('Type', 'device', 'Name', 'eth0')"
```

## Deploy the Prometheus-Skydive connector Pipeline

If desired, change the default settings in the yml file for the parameters:
```
analyzers
store.prom_sky_con.port
store.prom_sky_con.connection_timeout
```

Build and run the code in the prometheus-skydive-connector directory.

```
cd skydive-flow-exporter/prometheus
make static
prom_sky_con prom_sky_con.yml.default
```

To run as a container, perform:

```
docker run --privileged --pid=host --net=host -p 9100:9100 <prom_sky_con-docker-container>
```


## Prometheus setup

In the prometheus config file, add a stanza specifying the location and port used by the
prometheus-skydive connector.

```
  - job_name: 'skydive-connector'
    static_configs:
    - targets: ['xx.xx.xx.xx:9100']

```

## Verify Operation

Wait for 60 seconds and then check that we have data arriving in the prometheus GUI.
Look for a metric beginning with `skydive` such as `skydive_network_connection_total_bytes`.


