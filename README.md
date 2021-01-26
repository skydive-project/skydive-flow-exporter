# Overview

The Skydive Flow Exporter provides a framework for building pipelines which
extract flows from the Skydive Analyzer (via it WebSocket API), process them
and send the results upstream.

# Architecture

The pipeline is constructed of a sequence of steps:
- classify - tag flows according to some classification (such as ingress, egress)
- filter - filter out flows, possibly based on classification
- transform - transform Skydive Flow to any target record type
- mangle - process a group of transformed flows, enrich data, add new resulting
  flows, or delete redundent flows.
- storeheader - add a header marker to a group of flows to be stored
- store - handle grouping of flows to store object
- encode - encode store object (JSON, CSV, etc.)
- compress - compress encoded store object (GZIP, etc.) 
- account - record accounting information of store object
- write - write to target (S3, stdout, etc.)

# Deploy

We demonstrated deployment using IBM Security Advisor pipeline which extracts
flow data from Skydive, filters out external traffic, transforms flow records
by adding status information, and flow classification (egress, ingress,
internal), batches flows records together into streams (by time or count), then
encodes these blocks in JSON form, and compresses using GZIP and finally stores
as objects into any S3 compatible target.

## Deploy Skydive

Follow instructions on http://skydive.network/documentation/build to install prerequisites and prepare your machine to build the Skydive code.
Build and run Skydive:

```
make static
sudo $(which skydive) allinone -c etc/skydive.yml.default
```

## Setup Minio

For running tests on a local machine, setup a local Minio object store.

First run the minio server:

```
wget https://dl.min.io/server/minio/release/linux-amd64/minio
chmod +x minio
mkdir /tmp/data
export MINIO_VOLUMES="/var/lib/minio"
export MINIO_ACCESS_KEY=user
export MINIO_SECRET_KEY=password
./minio server /tmp/data
```

Next create the bucket (as destination location for flows):

```
wget https://dl.minio.io/client/mc/release/linux-amd64/mc
chmod +x mc
./mc config host add local http://localhost:9000 user password --api S3v4
./mc rm --recursive --force local/bucket
./mc rb --force local/bucket
./mc mb --ignore-existing local/bucket
```

## Deploy the Pipeline

Build and run the pipeline in the secadvisor directory:

```
cd secadvisor
make static
./secadvisor secadvisor.yml.default
```

secadvisor exectuable might be found in ~/go/bin/

## Create Capture

Setup capture on every interface:

```
skydive client capture create \
	--gremlin "G.V().Has('Type', 'device', 'Name', 'eth0')" \
	--type pcap \
	-c etc/skydive.yml.default
```

## Inject Traffic

```
skydive client inject-packet create --count 1000 --type tcp4 \
	--src "G.V().Has('Type', 'device', 'Name', 'eth0')" \
	--dst "G.V().Has('Type', 'device', 'Name', 'eth0')" \
	-c etc/skydive.yml.default
```

## Verify Operation

Wait for 60 seconds and then check that we have object accumulate in the bucket
as a result of the operation.

```
./mc ls --recursive local/bucket
```

# Development

You can either build a pipeline maintained within the current repository, or
you can create a pipeline external to this repository. A simple pipeline that
can be used as a starting point is the dummy pipeline.

