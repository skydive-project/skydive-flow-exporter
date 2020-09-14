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
 * This is a skeleton program to export skydive flow information to prometheus.
 * For each captured flow, we export the total number of bytes transferred on that flow.
 * Flows that have been inactive for some time are removed from the report.
 * Users may use the enclosed example as a base upon which to report additional skydive metrics through prometheus.
 */

package pkg

import (
	"container/list"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/skydive-project/skydive-flow-exporter/core"
	"github.com/skydive-project/skydive/flow"
	"github.com/skydive-project/skydive/logging"
)

// Connections with no traffic for timeout period (in seconds) are no longer reported
const defaultConnectionTimeout = 60

// Run the cleanup process every cleanupTime seconds
const cleanupTime = 120

const (
	DirectionItoT string = "initiator_to_target"
	DirectionTtoI string = "target_to_initiator"
)

// data to be exported to prometheus
// one data item per tuple
var (
	bytesSent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "skydive_network_connection_total_bytes",
			Help: "Number of bytes that have been transmmitted on this connection",
		},
		[]string{"initiator_ip", "target_ip", "initiator_port", "target_port", "direction", "node_tid"},
	)
)

// we maintain a cache to keep track of connections that have been active/inactive
// items of the cache are linked together, and an entry that is used is placed at the end of the list.
// items from the beginning of the list whose timeStamp has expired are deleted.
type cacheEntry struct {
	label1    prometheus.Labels
	label2    prometheus.Labels
	timeStamp int64
	e         *list.Element
	key       string
}

type myCache map[string]*cacheEntry

type storePrometheus struct {
	pipeline          *core.Pipeline // not currently used
	port              string
	connectionCache   myCache
	connectionList    *list.List
	connectionTimeout int64
}

// SetPipeline setup; called by core/pipeline.NewPipeline
func (s *storePrometheus) SetPipeline(pipeline *core.Pipeline) {
	s.pipeline = pipeline
}

// NewLabel helper function to create proper prometheus Label syntax
func NewLabel(initiator_ip, target_ip, initiator_port, target_port, direction, node_tid string) prometheus.Labels {
	label := prometheus.Labels{
		"initiator_ip":   initiator_ip,
		"target_ip":      target_ip,
		"initiator_port": initiator_port,
		"target_port":    target_port,
		"direction":      direction,
		"node_tid":       node_tid,
	}
	return label
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
			var cEntry *cacheEntry
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
				label1 = NewLabel(initiator_ip, target_ip, initiator_port, target_port, DirectionItoT, node_tid)
				label2 = NewLabel(initiator_ip, target_ip, initiator_port, target_port, DirectionTtoI, node_tid)
				cEntry = &cacheEntry{
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
			return
		}
		c := e.Value.(*cacheEntry)
		if c.timeStamp > expireTime {
			// no more expired items
			return
		}

		// clean up the entry
		logging.GetLogger().Debugf("secs = %s, deleting %s", secs, c.label1)
		bytesSent.Delete(c.label1)
		bytesSent.Delete(c.label2)
		delete(s.connectionCache, c.key)
		s.connectionList.Remove(e)
	}
}

func (s *storePrometheus) cleanupExpiredEntriesLoop() {
	for true {
		s.cleanupExpiredEntries()
		time.Sleep(cleanupTime * time.Second)
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
	s := &storePrometheus{
		port:              ":" + port_id,
		connectionCache:   make(map[string]*cacheEntry),
		connectionList:    list.New(),
		connectionTimeout: connectionTimeout,
	}
	return s, nil
}

// NewStorePrometheusWrapper returns a new interface for storing flows info to prometheus and starts the interface
func NewStorePrometheusWrapper(cfg *viper.Viper) (interface{}, error) {
	s, err := NewStorePrometheus(cfg)
	if err != nil {
		return nil, err
	}
	go startPrometheusInterface(s)
	go s.cleanupExpiredEntriesLoop()
	return s, nil
}
