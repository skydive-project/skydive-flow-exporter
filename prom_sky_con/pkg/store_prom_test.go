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

package pkg

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/skydive-project/skydive-flow-exporter/core"
	"github.com/skydive-project/skydive/config"
	"github.com/skydive-project/skydive/flow"
)

const testConfig = `---
pipeline:
  store:
    type: prom_sky_con
    prom_sky_con:
      port: 9100
      connection_timeout: 1
`

const initiator_ip = "192.168.0.5"
const target_ip = "173.194.40.147"
const initiator_port = 47838
const target_port = 80
const node_tid = "probe-tid"
const protocol = flow.FlowProtocol_TCP
const host_name = "host1"
const interface_name = "host1_name"
const interface_type = "face_eth"
const ipv6 = "10:20:30:40:50:60"

func initConfig(conf string) error {
	f, _ := ioutil.TempFile("", "store_prom_test")
	b := []byte(conf)
	f.Write(b)
	f.Close()
	err := config.InitConfig("file", []string{f.Name()})
	os.Remove(f.Name())
	return err
}

func assertEqual(t *testing.T, expected, actual interface{}) {
	if expected != actual {
		msg := "Equal assertion failed"
		_, file, no, ok := runtime.Caller(1)
		if ok {
			msg += fmt.Sprintf(" on %s:%d", file, no)
		}
		t.Fatalf("%s: (expected: %v, actual: %v)", msg, expected, actual)
	}
}

func getFlow() *flow.Flow {
	t, _ := time.Parse(time.RFC3339, "2019-01-01T10:20:30Z")
	start := flow.UnixMilli(t)
	return &flow.Flow{
		UUID:        "66724f5d-718f-47a2-93a7-c807cd54241e",
		LayersPath:  "Ethernet/IPv4/TCP",
		Application: "TCP",
		Link: &flow.FlowLayer{
			Protocol: flow.FlowProtocol_ETHERNET,
			A:        "fa:16:3e:29:e0:82",
			B:        "fa:16:3e:96:06:e8",
		},
		Network: &flow.FlowLayer{
			Protocol: flow.FlowProtocol_IPV4,
			A:        initiator_ip,
			B:        target_ip,
		},
		Transport: &flow.TransportLayer{
			Protocol: flow.FlowProtocol_TCP,
			A:        initiator_port,
			B:        target_port,
		},
		Metric: &flow.FlowMetric{
			ABPackets: 6,
			ABBytes:   516,
			BAPackets: 4,
			BABytes:   760,
			RTT:       1234567,
		},
		Start:        start,
		Last:         start,
		NodeTID:      node_tid,
		L3TrackingID: "66724f5d-718f-47a2-93a7-c807cd54241e",
	}
}

func TestStorePrometheus(t *testing.T) {
	initConfig(testConfig)
	cfg := config.GetConfig().Viper
	s, err := NewStorePrometheus(cfg)
	assertEqual(t, err, nil)
	f := getFlow()
	flows := make(map[core.Tag][]interface{})
	fList := make([]interface{}, 0)
	fList = append(fList, f)
	flows["other"] = fList
	s.StoreFlows(flows)

	// verify that the flow is reported to prometheus
	label1 := NewFlowLabel(initiator_ip, target_ip, strconv.FormatInt(initiator_port, 10), strconv.FormatInt(target_port, 10), DirectionItoT, node_tid, protocol)
	gaugeA, err := bytesSent.GetMetricWith(label1)
	bytesA := testutil.ToFloat64(gaugeA)
	assertEqual(t, nil, err)
	assertEqual(t, float64(f.Metric.ABBytes), bytesA)
	label2 := NewFlowLabel(initiator_ip, target_ip, strconv.FormatInt(initiator_port, 10), strconv.FormatInt(target_port, 10), DirectionTtoI, node_tid, protocol)
	gaugeB, err := bytesSent.GetMetricWith(label2)
	assertEqual(t, nil, err)
	bytesB := testutil.ToFloat64(gaugeB)
	assertEqual(t, float64(f.Metric.BABytes), bytesB)

	gaugeA2, err := packetsSent.GetMetricWith(label1)
	packetsA := testutil.ToFloat64(gaugeA2)
	assertEqual(t, nil, err)
	assertEqual(t, float64(f.Metric.ABPackets), packetsA)
	gaugeB2, err := packetsSent.GetMetricWith(label2)
	packetsB := testutil.ToFloat64(gaugeB2)
	assertEqual(t, nil, err)
	assertEqual(t, float64(f.Metric.BAPackets), packetsB)

	gaugeR, err := rtt.GetMetricWith(label1)
	rtt := testutil.ToFloat64(gaugeR)
	assertEqual(t, nil, err)
	assertEqual(t, float64(f.Metric.RTT), rtt)

	// verify entry is in cache
	entriesMap := s.connectionCache
	assertEqual(t, 1, len(entriesMap))
	_, found := s.connectionCache[f.L3TrackingID]
	assertEqual(t, true, found)

	// wait a couple seconds so that the entry will expire
	time.Sleep(2 * time.Second)
	s.cleanupExpiredEntries()
	entriesMap = s.connectionCache
	assertEqual(t, 0, len(entriesMap))
}

func getInterface() map[string]interface{} {
	metrics := map[string]float64 {
		"RxBytes": float64(10),
		"TxBytes": float64(20),
		"RxPackets": float64(2),
		"TxPackets": float64(3),
		"RxErrors": float64(1),
	}
	ip4 := [](interface{}) { initiator_ip }
	ip6 := [](interface{}) { ipv6 }
	metadata := map[string]interface{} {
		"Name": interface_name,
		"Type": interface_type,
		"TID": node_tid,
		"IPV4": ip4,
		"IPV6": ip6,
		"Metric": metrics,
	}
	test_interface := map[string]interface{} {
		"Host": host_name,
		"Metadata": metadata,
	}
	return test_interface
}

func TestStoreInterfaceMetrics(t *testing.T) {
	initConfig(testConfig)
	cfg := config.GetConfig().Viper
	s, err := NewStorePrometheus(cfg)
	assertEqual(t, err, nil)
	e := getInterface()
	eList := make([]map[string]interface{}, 0)
	eList = append(eList, e)
	s.StoreInterfaceMetrics(eList)

	// verify that the interface info is reported to prometheus
	label := NewInterfaceLabel(interface_name, host_name, interface_type, initiator_ip, ipv6,  node_tid)

	gaugeA, err := rxBytes.GetMetricWith(label)
	bytesA := testutil.ToFloat64(gaugeA)
	assertEqual(t, nil, err)
	metadata := e["Metadata"].(map[string]interface{})
	metrics := metadata["Metric"].(map[string]float64)
	assertEqual(t, metrics["RxBytes"], bytesA)

	gaugeA, err = txBytes.GetMetricWith(label)
	bytesA = testutil.ToFloat64(gaugeA)
	assertEqual(t, nil, err)
	metadata = e["Metadata"].(map[string]interface{})
	metrics = metadata["Metric"].(map[string]float64)
	assertEqual(t, metrics["TxBytes"], bytesA)

	gaugeA, err = rxDropped.GetMetricWith(label)
	bytesA = testutil.ToFloat64(gaugeA)
	assertEqual(t, nil, err)
	metadata = e["Metadata"].(map[string]interface{})
	metrics = metadata["Metric"].(map[string]float64)
	assertEqual(t, metrics["RxDropped"], bytesA)

	// verify entry is in cache
	entriesMap := s.interfCache
	assertEqual(t, 1, len(entriesMap))
	_, found := s.interfCache[node_tid]
	assertEqual(t, true, found)

	// wait a couple seconds so that the entry will expire
	time.Sleep(2 * time.Second)
	s.cleanupExpiredEntries()
	entriesMap = s.interfCache
	assertEqual(t, 0, len(entriesMap))
}
