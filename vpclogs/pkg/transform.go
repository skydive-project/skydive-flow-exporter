/*
 * Copyright (C) 2019 IBM, Inc.
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
	"strings"
	"time"

	cache "github.com/pmylund/go-cache"
	"github.com/spf13/viper"

	"github.com/skydive-project/skydive-flow-exporter/core"
	"github.com/skydive-project/skydive/api/client"
	"github.com/skydive-project/skydive/flow"
	"github.com/skydive-project/skydive/gremlin"
	"github.com/skydive-project/skydive/logging"
)

const (
	VpcActionReject string = "R"
	VpcActionAccept string = "A"
)

// IP protocol type (for example, TCP = 6; UDP = 17, ICMP = 1)
const (
	VpcProtocolICMP uint8 = 1
	VpcProtocolTCP  uint8 = 6
	VpcProtocolUDP  uint8 = 17
)

const (
	VpcDirectionInbound  string = "inbound"
	VpcDirectionOutbound string = "outbound"
	VpcDirectionEmpty    string = ""
)

// VpclogsFlow represents a vpc flow log entry
// we have several fields for intermediate values, but which are not part of the final output
type VpclogsFlow struct {
	Flow                           *flow.Flow       `json:"-"`
	LastUpdateMetric               *flow.FlowMetric `json:"-"`
	Metric                         *flow.FlowMetric `json:"-"`
	Protocol                       string           `json:"-"`
	StartTime                      string           `json:"start_time,omitempty"`
	EndTime                        string           `json:"end_time,omitempty"`
	ConnectionStartTime            string           `json:"connection_start_time,omitempty"`
	Direction                      string           `json:"direction,omitempty"`
	Action                         string           `json:"action,omitempty"`
	InitiatorIp                    string           `json:"initiator_ip,omitempty"`
	TargetIp                       string           `json:"target_ip,omitempty"`
	InitiatorPort                  int64            `json:"initiator_port,omitempty"`
	TargetPort                     int64            `json:"target_port,omitempty"`
	TransportProtocol              uint8            `json:"transport_protocol,omitempty"`
	EtherType                      string           `json:"ether_type,omitempty"`
	WasInitiated                   bool             `json:"was_initiated"`
	WasTerminated                  bool             `json:"was_terminated"`
	CumulativeBytesFromInitiator   int64            `json:"cumulative_bytes_from_initiator,omitempty"`
	CumulativePacketsFromInitiator int64            `json:"cumulative_packets_from_initiator,omitempty"`
	CumulativeBytesFromTarget      int64            `json:"cumulative_bytes_from_target,omitempty"`
	CumulativePacketsFromTarget    int64            `json:"cumulative_packets_from_target,omitempty"`
	BytesFromInitiator             int64            `json:"bytes_from_initiator,omitempty"`
	PacketsFromInitiator           int64            `json:"packets_from_initiator,omitempty"`
	BytesFromTarget                int64            `json:"bytes_from_target,omitempty"`
	PacketsFromTarget              int64            `json:"packets_from_target,omitempty"`
}

// VpclogsFlowTransformer is a custom transformer for flows
type vpclogsFlowTransformer struct {
	interfaceIpCache *cache.Cache
	gremlinClient    *client.GremlinQueryHelper
}

// NewTransform creates a new flow transformer for vpclogs
func NewTransform(cfg *viper.Viper) (interface{}, error) {
	gremlinClient := client.NewGremlinQueryHelper(core.CfgAuthOpts(cfg))
	return &vpclogsFlowTransformer{
		interfaceIpCache: cache.New(10*time.Minute, 10*time.Minute),
		gremlinClient:    gremlinClient,
	}, nil
}

// TIDtoIP maps a TID into its IP addresses
// We save the list of IP addresses in a local cache
// Note - Not all node types have IP addresses, in which case we save an empty list of addresses
func (ft *vpclogsFlowTransformer) TIDtoIP(TID string) ([]string, error) {
	result, ok := ft.interfaceIpCache.Get(TID)
	if !ok {
		var ipAddrList []string
		var ipAddrList2 []string
		node, err := ft.gremlinClient.GetNode(gremlin.G.V().Has("TID", TID))
		if err != nil {
			emptyList := make([]string, 0)
			return emptyList, err
		}
		ipAddrList, err = node.GetFieldStringList("IPV4")
		ipAddrList2 = make([]string, 0)
		if err == nil {
			// strip the subnet masks from the addresses
			for _, s := range ipAddrList {
				tmp := strings.Split(s, "/")
				ipAddr := tmp[0]
				ipAddrList2 = append(ipAddrList2, ipAddr)
			}
		}
		ft.interfaceIpCache.Set(TID, ipAddrList2, cache.DefaultExpiration)
		return ipAddrList, nil
	}
	return result.([]string), nil
}

// setDirection sets the Dirction field in the VpclogsFlow structure
// If source address matches the IP address of the interface, the flow is outgoing.
// Sometimes the IP address recorded in a node includes a subnet mask, so we check that the prefix matches the IP
func (ft *vpclogsFlowTransformer) setDirection(v *VpclogsFlow) {
	ipAddrList, _ := ft.TIDtoIP(v.Flow.NodeTID)
	for _, s := range ipAddrList {
		if s == v.InitiatorIp {
			v.Direction = VpcDirectionOutbound
			return
		}
		if s == v.TargetIp {
			v.Direction = VpcDirectionInbound
			return
		}
	}
	v.Direction = VpcDirectionEmpty
}

// SetAction sets the action field in the vpc log structure; expected to be called by an external function
func (v *VpclogsFlow) SetAction(action string) {
	v.Action = action
}

// DeriveExternalizedFields builds the fields that are visible in the vpc flow logs.
// These values are derived from values stored in the VpclogsFlow structure (not from the original flows)
// since these fields may have been updated by additional processing (e.g. by action).
func (v *VpclogsFlow) DeriveExternalizedFields() {
	v.StartTime = time.Unix(v.LastUpdateMetric.Start/1000, 0).UTC().Format(time.RFC3339)
	v.EndTime = time.Unix(v.LastUpdateMetric.Last/1000, 0).UTC().Format(time.RFC3339)
	// In the meantime, EtherType is hard-coded
	v.EtherType = "IPv4"
	v.ConnectionStartTime = time.Unix(v.Metric.Start/1000, 0).UTC().Format(time.RFC3339)
	v.CumulativeBytesFromInitiator = v.Metric.ABBytes
	v.CumulativePacketsFromInitiator = v.Metric.ABPackets
	v.CumulativeBytesFromTarget = v.Metric.BABytes
	v.CumulativePacketsFromTarget = v.Metric.BAPackets
	v.BytesFromInitiator = v.LastUpdateMetric.ABBytes
	v.PacketsFromInitiator = v.LastUpdateMetric.ABPackets
	v.BytesFromTarget = v.LastUpdateMetric.BABytes
	v.PacketsFromTarget = v.LastUpdateMetric.BAPackets

	// convert prtotocol type from string to int
	if v.Protocol == "TCP" {
		v.TransportProtocol = VpcProtocolTCP
	} else if v.Protocol == "UDP" {
		v.TransportProtocol = VpcProtocolUDP
	} else if v.Protocol == "ICMP" {
		v.TransportProtocol = VpcProtocolICMP
	}
}

// Transform transforms a flow from skydive format to vpc logs format before being stored
func (ft *vpclogsFlowTransformer) Transform(f *flow.Flow) interface{} {

	// flows of type ARP are ignored
	if f.Transport == nil {
		return nil
	}
	v := &VpclogsFlow{
		Flow:             f,
		Protocol:         f.Transport.Protocol.String(),
		LastUpdateMetric: f.LastUpdateMetric,
		Metric:           f.Metric,
		InitiatorIp:      f.Network.A,
		TargetIp:         f.Network.B,
		InitiatorPort:    f.Transport.A,
		TargetPort:       f.Transport.B,
	}

	ft.setDirection(v)

	v.WasTerminated = f.TCPMetric.ABFinStart != 0 || f.TCPMetric.BAFinStart != 0 || f.TCPMetric.ABRstStart != 0 || f.TCPMetric.BARstStart != 0
	// for some reason the line below does not provide the desired result, so we do a different hack in the meantime
	//v.WasInitiated = f.TCPMetric.ABSynStart != 0 || f.TCPMetric.BASynStart != 0
	v.WasInitiated = v.Metric.ABBytes == v.LastUpdateMetric.ABBytes || v.Metric.BABytes == v.LastUpdateMetric.BABytes

	v.DeriveExternalizedFields()
	logging.GetLogger().Debugf("original flow = %s \n transformed flow = %s", f, v)
	return v
}
