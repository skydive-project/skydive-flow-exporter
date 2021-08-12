/*
 * Copyright (C) 2020 IBM, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy ofthe License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specificlanguage governing permissions and
 * limitations under the License.
 *
 */

/*
 * This is a program to export skydive metrics to prometheus.
 * For each captured flow, we export the total number of bytes transferred on that flow.
 * Flows that have been inactive for some time are removed from the report.
 * In addition to per flow metrics, metrics are reported for network interfaces.
 */

package pkg

import (
	"container/list"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/skydive-project/skydive-flow-exporter/core"
	"github.com/skydive-project/skydive/api/client"
	"github.com/skydive-project/skydive/flow"
	"github.com/skydive-project/skydive/graffiti/logging"
)

// Connections with no traffic for timeout period (in seconds) are no longer reported
const defaultConnectionTimeout = 60

// Run the cleanup process every cleanupTime seconds
const cleanupTime = 120

// Collect the interface metrics every interfaceMetricsTime seconds
const interfaceMetricsTime = 20

const (
	DirectionItoT string = "initiator_to_target"
	DirectionTtoI string = "target_to_initiator"
)

var connectionLabels = []string{"initiator_ip", "target_ip", "initiator_port", "target_port", "direction", "node_tid", "protocol"}
var interfaceLabels = []string{"interface_name", "host", "interface_type", "ipv4", "ipv6", "node_tid"}

// data to be exported to prometheus
// one data item per tuple
var (
	bytesSent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_connection_total_bytes",
			Help: "Number of bytes that have been transmmitted on this connection",
		},
		connectionLabels,
	)
	packetsSent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_connection_total_packets",
			Help: "Number of packets that have been transmmitted on this connection",
		},
		connectionLabels,
	)
	rtt = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_connection_rtt_nanoseconds",
			Help: "Round Trip Time of packets that have been transmitted on this connection",
		},
		connectionLabels,
	)
	rxBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_interface_rx_bytes",
			Help: "Number of bytes that have been received on this interface",
		},
		interfaceLabels,
	)
	txBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_interface_tx_bytes",
			Help: "Number of bytes that have been transmitted by this interface",
		},
		interfaceLabels,
	)
	rxPackets = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_interface_rx_packets",
			Help: "Number of packets that have been received on this interface",
		},
		interfaceLabels,
	)
	txPackets = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_interface_tx_packets",
			Help: "Number of packets that have been transmitted by this interface",
		},
		interfaceLabels,
	)
	rxDropped = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_interface_rx_dropped",
			Help: "Number of received packets that have been dropped by this interface",
		},
		interfaceLabels,
	)
	txDropped = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_interface_tx_dropped",
			Help: "Number of transmitted packets that have been dropped on this interface",
		},
		interfaceLabels,
	)
	rxErrors = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_interface_rx_errors",
			Help: "Number of errors on received packets on this interface",
		},
		interfaceLabels,
	)
	txErrors = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_interface_tx_errors",
			Help: "Number of errors on transmitted packets on this interface",
		},
		interfaceLabels,
	)
	rxMissedErrors = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_interface_rx_missed_errors",
			Help: "Number of missed errors on received packets on this interface",
		},
		interfaceLabels,
	)
	txMissedErrors = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_interface_tx_missed_errors",
			Help: "Number of missed errors on transmitted packets on this interface",
		},
		interfaceLabels,
	)
	collisions = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_interface_collisions",
			Help: "Number of collisions reported on this interface",
		},
		interfaceLabels,
	)
)

// we maintain a cache to keep track of connections that have been active/inactive
// items of the cache are linked together, and an entry that is used is placed at the end of the list.
// items from the beginning of the list whose timeStamp has expired are deleted.
type flowCacheEntry struct {
	label1    prometheus.Labels
	label2    prometheus.Labels
	timeStamp int64
	e         *list.Element
	key       string
}

type flowCache map[string]*flowCacheEntry

type interfaceCacheEntry struct {
	label     prometheus.Labels
	timeStamp int64
	e         *list.Element
	key       string
}

type interfaceCache map[string]*interfaceCacheEntry

type storePrometheus struct {
	pipeline          *core.Pipeline // not currently used
	port              string
	connectionCache   flowCache
	connectionList    *list.List
	interfCache       interfaceCache
	interfaceList     *list.List
	connectionTimeout int64
	gremlinClient     *client.GremlinQueryHelper
}

// SetPipeline setup; called by core/pipeline.NewPipeline
func (s *storePrometheus) SetPipeline(pipeline *core.Pipeline) {
	s.pipeline = pipeline
}

// NewFlowLabel helper function to create proper prometheus Label syntax
func NewFlowLabel(initiator_ip, target_ip, initiator_port, target_port, direction, node_tid string, protocol flow.FlowProtocol) prometheus.Labels {
	label := prometheus.Labels{
		"initiator_ip":   initiator_ip,
		"target_ip":      target_ip,
		"initiator_port": initiator_port,
		"target_port":    target_port,
		"direction":      direction,
		"node_tid":       node_tid,
		"protocol":       protocol.String(),
	}
	return label
}

// NewInterfaceLabel helper function to create proper prometheus Label syntax
func NewInterfaceLabel(interface_name, host, interface_type, ipv4, ipv6, node_tid string) prometheus.Labels {
	label := prometheus.Labels{
		"interface_name": interface_name,
		"host":           host,
		"interface_type": interface_type,
		"ipv4":           ipv4,
		"ipv6":           ipv6,
		"node_tid":       node_tid,
	}
	return label
}

// StoreInterfaceMetrics - parse list of interface metrics received from skydive
func (s *storePrometheus) StoreInterfaceMetrics(interface_metrics []map[string]interface{}) {
	secs := time.Now().Unix()
	for _, interface_entry := range interface_metrics {
		var ipv4, ipv6 string
		var metrics map[string]float64

		// for each interface entry, extract all the relevant information
		host := interface_entry["Host"].(string)
		metadata := interface_entry["Metadata"].(map[string]interface{})
		interface_name := metadata["Name"].(string)
		interface_type := metadata["Type"].(string)
		node_tid := metadata["TID"].(string)

		// not all network interfaces have an IPV4 address defined
		ip, ok := metadata["IPV4"]
		if !ok {
			ipv4 = ""
		} else {
			ipv4 = ip.([]interface{})[0].(string)
		}
		ip, ok = metadata["IPV6"]
		if !ok {
			ipv6 = ""
		} else {
			ipv6 = ip.([]interface{})[0].(string)
		}

		// metrics are reported only if non-zero; fill in missing metrics
		metrics, ok = metadata["Metric"].(map[string]float64)
		if !ok {
			metrics = make(map[string]float64)
			metrics["RxBytes"] = float64(0)
			metrics["TxBytes"] = float64(0)
			metrics["RxPackets"] = float64(0)
			metrics["TxPackets"] = float64(0)
			metrics["RxErrors"] = float64(0)
			metrics["TxErrors"] = float64(0)
			metrics["RxMissedErrors"] = float64(0)
			metrics["TxMissedErrors"] = float64(0)
			metrics["RxDropped"] = float64(0)
			metrics["TxDropped"] = float64(0)
			metrics["Collisions"] = float64(0)
		}
		rx_bytes, ok := metrics["RxBytes"]
		if !ok {
			rx_bytes = 0
		}
		tx_bytes, ok := metrics["TxBytes"]
		if !ok {
			tx_bytes = 0
		}
		rx_packets, ok := metrics["RxPackets"]
		if !ok {
			rx_packets = 0
		}
		tx_packets, ok := metrics["TxPackets"]
		if !ok {
			tx_packets = 0
		}
		rx_errors, ok := metrics["RxErrors"]
		if !ok {
			rx_errors = 0
		}
		tx_errors, ok := metrics["TxErrors"]
		if !ok {
			tx_errors = 0
		}
		rx_missed_errors, ok := metrics["RxMissedErrors"]
		if !ok {
			rx_missed_errors = 0
		}
		tx_missed_errors, ok := metrics["TxMissedErrors"]
		if !ok {
			tx_missed_errors = 0
		}
		rx_dropped, ok := metrics["RxDropped"]
		if !ok {
			rx_dropped = 0
		}
		tx_dropped, ok := metrics["TxDropped"]
		if !ok {
			tx_dropped = 0
		}
		collisions_, ok := metrics["Collisions"]
		if !ok {
			collisions_ = 0
		}

		// place item into cache
		var cEntry *interfaceCacheEntry
		var label prometheus.Labels

		cEntry, ok = s.interfCache[node_tid]
		if ok {
			// item already exists in cache; update the element and move to end of list
			cEntry.timeStamp = secs
			label = cEntry.label
			// move to end of list
			s.interfaceList.MoveToBack(cEntry.e)
		} else {
			// create new entry for cache
			label = NewInterfaceLabel(interface_name, host, interface_type, ipv4, ipv6, node_tid)
			cEntry = &interfaceCacheEntry{
				label:     label,
				timeStamp: secs,
				key:       node_tid,
			}
			// place at end of list
			e := s.interfaceList.PushBack(cEntry)
			cEntry.e = e
			s.interfCache[node_tid] = cEntry
		}

		// report the metrics to prometheus
		logging.GetLogger().Debugf("rx_bytes = %d, label = %s", rx_bytes, label)
		rxBytes.With(label).Set(rx_bytes)
		txBytes.With(label).Set(tx_bytes)
		rxPackets.With(label).Set(rx_packets)
		txPackets.With(label).Set(tx_packets)
		rxErrors.With(label).Set(rx_errors)
		txErrors.With(label).Set(tx_errors)
		rxMissedErrors.With(label).Set(rx_missed_errors)
		txMissedErrors.With(label).Set(tx_missed_errors)
		rxDropped.With(label).Set(rx_dropped)
		txDropped.With(label).Set(tx_dropped)
		collisions.With(label).Set(collisions_)
	}
}

// InterfaceMetrics - obtain and process list of interface metrics from skydive
func (s *storePrometheus) InterfaceMetrics() {
	// obtain skydive nodes of all network interfaces
	queryString := "G.V().Has('EncapType', 'ether')"
	result, err := s.gremlinClient.Query(queryString)
	if err != nil {
		logging.GetLogger().Errorf("InterfaceMetrics: gremlin query error; err = %s, result = %s", err, result)
		return
	} else if len(result) == 0 {
		logging.GetLogger().Errorf("InterfaceMetrics: gremlin empty query")
		return
	}
	// covert query (json text) result to dictionaries
	// result is a list of dictionaries, one list entry per network interface.
	var result2 []map[string]interface{}
	err = json.Unmarshal(result, &result2)
	if err != nil {
		logging.GetLogger().Errorf("InterfaceMetrics: gremlin marshalling error; err = %s, result2 = %s", err, result2)
		return
	}
	s.StoreInterfaceMetrics(result2)
}

// StoreFlows store flows info in memory, before being shipped out
// For each flow reported, prepare prometheus entries, one for each direction of data flow.
func (s *storePrometheus) StoreFlows(flows map[core.Tag][]interface{}) error {
	secs := time.Now().Unix()
	for _, val := range flows {
		for _, i := range val {
			f := i.(*flow.Flow)
			if f.Transport == nil {
				continue
			}
			logging.GetLogger().Debugf("flow = %s", f)

			var ok bool
			var cEntry *flowCacheEntry
			var label1, label2 prometheus.Labels

			cEntry, ok = s.connectionCache[f.L3TrackingID]
			if ok {
				// item already exists in cache; update the element and move to end of list
				cEntry.timeStamp = secs
				label1 = cEntry.label1
				label2 = cEntry.label2
				// move to end of list
				s.connectionList.MoveToBack(cEntry.e)
			} else {
				// create new entry for cache
				initiator_ip := f.Network.A
				target_ip := f.Network.B
				initiator_port := strconv.FormatInt(f.Transport.A, 10)
				target_port := strconv.FormatInt(f.Transport.B, 10)
				node_tid := f.NodeTID
				protocol := f.Transport.Protocol
				label1 = NewFlowLabel(initiator_ip, target_ip, initiator_port, target_port, DirectionItoT, node_tid, protocol)
				label2 = NewFlowLabel(initiator_ip, target_ip, initiator_port, target_port, DirectionTtoI, node_tid, protocol)
				cEntry = &flowCacheEntry{
					label1:    label1,
					label2:    label2,
					timeStamp: secs,
					key:       f.L3TrackingID,
				}
				// place at end of list
				e := s.connectionList.PushBack(cEntry)
				cEntry.e = e
				s.connectionCache[f.L3TrackingID] = cEntry
			}
			// post the info to prometheus
			bytesSent.With(label1).Set(float64(f.Metric.ABBytes))
			bytesSent.With(label2).Set(float64(f.Metric.BABytes))
			packetsSent.With(label1).Set(float64(f.Metric.ABPackets))
			packetsSent.With(label2).Set(float64(f.Metric.BAPackets))
			rtt.With(label1).Set(float64(f.Metric.RTT))
		}
	}
	return nil
}

// cleanupExpiredEntries - any entry that has expired should be removed from the prometheus reporting and cache
func (s *storePrometheus) cleanupExpiredEntries() {
	secs := time.Now().Unix()
	expireTime := secs - s.connectionTimeout
	// go through the list until we reach recently used connections
	for true {
		e := s.connectionList.Front()
		if e == nil {
			break
		}
		c := e.Value.(*flowCacheEntry)
		if c.timeStamp > expireTime {
			// no more expired items
			break
		}

		// clean up the entry
		logging.GetLogger().Debugf("secs = %s, deleting %s", secs, c.label1)
		bytesSent.Delete(c.label1)
		bytesSent.Delete(c.label2)
		packetsSent.Delete(c.label1)
		packetsSent.Delete(c.label2)
		rtt.Delete(c.label1)
		delete(s.connectionCache, c.key)
		s.connectionList.Remove(e)
	}

	// go through the list until we reach recently used interfaces
	for true {
		e := s.interfaceList.Front()
		if e == nil {
			break
		}
		c := e.Value.(*interfaceCacheEntry)
		if c.timeStamp > expireTime {
			// no more expired items
			break
		}

		// clean up the entry
		logging.GetLogger().Debugf("secs = %s, deleting %s", secs, c.label)
		rxBytes.Delete(c.label)
		txBytes.Delete(c.label)
		rxPackets.Delete(c.label)
		txPackets.Delete(c.label)
		rxErrors.Delete(c.label)
		txErrors.Delete(c.label)
		rxMissedErrors.Delete(c.label)
		txMissedErrors.Delete(c.label)
		rxDropped.Delete(c.label)
		txDropped.Delete(c.label)
		collisions.Delete(c.label)

		delete(s.interfCache, c.key)
		s.interfaceList.Remove(e)
	}
}

func (s *storePrometheus) cleanupExpiredEntriesLoop() {
	for true {
		s.cleanupExpiredEntries()
		time.Sleep(cleanupTime * time.Second)
	}
}

// interfaceMericsLoop - independent running loop to periodically obtain interface metrics
func (s *storePrometheus) interfaceMericsLoop() {
	for true {
		s.InterfaceMetrics()
		time.Sleep(interfaceMetricsTime * time.Second)
	}
}

// registerCollector - needed in order to send metrics to prometheus
func registerCollector(c prometheus.Collector) {
	prometheus.Register(c)
}

// startPrometheusInterface listens for prometheus resource usage requests
func startPrometheusInterface(s *storePrometheus) {

	// Metrics have to be registered to be exposed:
	registerCollector(bytesSent)
	registerCollector(packetsSent)
	registerCollector(rtt)
	registerCollector(rxBytes)
	registerCollector(txBytes)
	registerCollector(rxPackets)
	registerCollector(txPackets)
	registerCollector(rxErrors)
	registerCollector(txErrors)
	registerCollector(rxMissedErrors)
	registerCollector(txMissedErrors)
	registerCollector(rxDropped)
	registerCollector(txDropped)
	registerCollector(collisions)

	// The Handler function provides a default handler to expose metrics
	// via an HTTP server. "/metrics" is the usual endpoint for that.
	http.Handle("/metrics", promhttp.Handler())

	err := http.ListenAndServe(s.port, nil)
	if err != nil {
		logging.GetLogger().Errorf("error on http.ListenAndServe = %s", err)
		os.Exit(1)
	}
}

// NewStorePrometheus allocates and initializes the storePrometheus structure
func NewStorePrometheus(cfg *viper.Viper) (*storePrometheus, error) {
	// process user defined parameters from yml file
	port_id := cfg.GetString(core.CfgRoot + "store.prom_sky_con.port")
	if port_id == "" {
		logging.GetLogger().Errorf("prometheus skydive port missing in configuration file")
		return nil, fmt.Errorf("Failed to detect port number")
	}
	logging.GetLogger().Infof("prometheus skydive port = %s", port_id)
	connectionTimeout := cfg.GetInt64(core.CfgRoot + "store.prom_sky_con.connection_timeout")
	if connectionTimeout == 0 {
		connectionTimeout = defaultConnectionTimeout
	}
	logging.GetLogger().Infof("connection timeout = %d", connectionTimeout)

	// for interface metrics, we use a gremlinClient
	gremlinClient, err := client.NewGremlinQueryHelperFromConfig(core.CfgAuthOpts(cfg))
	if err != nil {
		return nil, err
	}

	s := &storePrometheus{
		port:              ":" + port_id,
		connectionCache:   make(map[string]*flowCacheEntry),
		connectionList:    list.New(),
		interfCache:       make(map[string]*interfaceCacheEntry),
		interfaceList:     list.New(),
		connectionTimeout: connectionTimeout,
		gremlinClient:     gremlinClient,
	}
	return s, nil
}

// NewStorePrometheusWrapper returns a new interface for storing flows info to prometheus and starts the interface
func NewStorePrometheusWrapper(cfg *viper.Viper) (interface{}, error) {
	s, err := NewStorePrometheus(cfg)
	if err != nil {
		return nil, err
	}
	// start independent threads to perform actions in parallel
	go startPrometheusInterface(s)
	go s.cleanupExpiredEntriesLoop()
	go s.interfaceMericsLoop()
	return s, nil
}
