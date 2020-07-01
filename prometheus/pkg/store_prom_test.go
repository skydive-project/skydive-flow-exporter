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
	"github.com/skydive-project/skydive/common"
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
	start := common.UnixMillis(t)
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
	s, err := NewStorePrometheusInternal(cfg)
	assertEqual(t, err, nil)
	f := getFlow()
	flows := make(map[core.Tag][]interface{})
	fList := make([]interface{}, 0)
	fList = append(fList, f)
	flows["other"] = fList
	s.StoreFlows(flows)

	// verify that the flow is reported to prometheus
	label1 := NewLabel(initiator_ip, target_ip, strconv.FormatInt(initiator_port, 10), strconv.FormatInt(target_port, 10), DirectionItoT, node_tid)
	gaugeA, err := bytesSent.GetMetricWith(label1)
	bytesA := testutil.ToFloat64(gaugeA)
	assertEqual(t, nil, err)
	assertEqual(t, float64(f.Metric.ABBytes), bytesA)
	label2 := NewLabel(initiator_ip, target_ip, strconv.FormatInt(initiator_port, 10), strconv.FormatInt(target_port, 10), DirectionTtoI, node_tid)
	gaugeB, err := bytesSent.GetMetricWith(label2)
	assertEqual(t, nil, err)
	bytesB := testutil.ToFloat64(gaugeB)
	assertEqual(t, float64(f.Metric.BABytes), bytesB)

	// verify entry is in cache
	entriesMap := s.connectionCache.Items()
	assertEqual(t, 1, len(entriesMap))
	_, found := s.connectionCache.Get(f.L3TrackingID)
	assertEqual(t, true, found)

	// wait a couple seconds so that the entry will expire
	time.Sleep(2 * time.Second)
	s.cleanupExpiredEntries()
	entriesMap = s.connectionCache.Items()
	assertEqual(t, 0, len(entriesMap))
}
