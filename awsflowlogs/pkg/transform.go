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
	"strconv"

	"github.com/spf13/viper"

	"github.com/skydive-project/skydive/flow"

	"github.com/skydive-project/skydive-flow-exporter/core"
)

const (
	accountID = "12345678"
)

type Action string

const (
	ActionReject Action = "REJECT"
	ActionAccept Action = "ACCEPT"
)

type LogStatus string

const (
	LogStatusOk       LogStatus = "OK"
	LogStatusNoData   LogStatus = "NODATA"
	LogStatusSkipData LogStatus = "SKIPDATA"
)

const (
	TCPFlagsFIN = 1
	TCPFlagsSYN = 2
	TCPFlagsRST = 4
	TCPFlagsACK = 16
)

const (
	TypeIPv4 = "IPv4"
	TypeIPv6 = "IPv6"
	TypeEFA  = "EFA" // Elastic Fabric Adapter.
)

type recordV2 struct {
	// The VPC Flow Logs version.
	Version int `csv:"version"`
	// The AWS account ID for the flow log.
	AccountID string `csv:"account-id"`
	// The ID of the network interface for which the traffic is recorded.
	InterfaceID string `csv:"interface-id"`
	// The source IPv4 or IPv6 address. The IPv4 address of the network
	// interface is always its private IPv4 address.
	SrcAddr string `csv:"srcadr"`
	// The destination IPv4 or IPv6 address. The IPv4 address of the
	// network interface is always its private IPv4 address.
	DstAddr string `csv:"dstaddr"`
	// The source port of the traffic.
	SrcPort int `csv:"srcport"`
	// The destination port of the traffic.
	DstPort int `csv:"dstport"`
	// The IANA protocol number of the traffic. For more information, see
	// Assigned Internet Protocol Numbers.
	Protocol int `csv:"protocol"`
	// The number of packets transferred during the capture window.
	Packets int64 `csv:"packets"`
	// The number of bytes transferred during the capture window.
	Bytes int64 `csv:"bytes"`
	// The time, in Unix seconds, of the start of the capture window.
	Start int64 `csv:"start"`
	// The time, in Unix seconds, of the end of the capture window.
	End int64 `csv:"end"`
	// The recorded traffic: ACCEPT if permitted: REJECT if not permitted
	// (by security groups of network ACLS).
	Action Action `csv:"action"`
	// The logging status: OK if logged normally; NODATA if no data during
	// cature window; SKIPDATA due to possible cap of traffic.
	LogStatus LogStatus `csv:"log-status"`
}

type recordV3 struct {
	// extends on v2 fields
	recordV2
	// The ID of the VPC that contains the network interface for which the
	// traffic is recorded.
	VpcID string `csv:"vpc-id"`
	// The ID of the subnet that contains the network interface for which
	// the traffic is recorded.
	SubnetID string `csv:"subnet-id"`
	// The ID of the instance that's associated with network interface for
	// which the traffic is recorded, if the instance is owned by you.
	// Returns a '-' symbol for a requester-managed network interface; for
	// example, the network interface for a NAT gateway.
	InstanceID string `csv:"instance-id"`
	// The bitmask value for select TCP flags.
	TCPFlags int `csv:"tcp-flags"`
	// The type of traffic.
	Type string `csv:"type"`
	// The packet-level (original) source IP address of the traffic.
	PktSrcAddr string `csv:"pkt-srcadr"`
	// The packet-level (original) destination IP address of the traffic.
	PktDstAddr string `csv:"pkt-dstaddr"`
}

type transform struct {
	version int
}

// NewTransform creates a new transformer
func NewTransform(cfg *viper.Viper) (interface{}, error) {
	return &transform{
		version: cfg.GetInt(core.CfgRoot + "transform.awsflowlogs.version"),
	}, nil
}

func getProtocol(f *flow.Flow) int {
	if f.Transport == nil {
		return 0
	}

	return int(f.Transport.Protocol.Value())
}

func getNetworkA(f *flow.Flow) string {
	if f.Network == nil {
		return ""
	}
	return f.Network.A
}

func getNetworkB(f *flow.Flow) string {
	if f.Network == nil {
		return ""
	}
	return f.Network.B
}

func getTransportA(f *flow.Flow) int {
	if f.Transport == nil {
		return 0
	}
	return int(f.Transport.A)
}

func getTransportB(f *flow.Flow) int {
	if f.Transport == nil {
		return 0
	}
	return int(f.Transport.B)
}

func getType(f *flow.Flow) string {
	if f.Network != nil {
		switch f.Network.Protocol {
		case flow.FlowProtocol_IPV4:
			return TypeIPv4
		case flow.FlowProtocol_IPV6:
			return TypeIPv6
		}

	}
	return ""
}

// Transform transforms a flow before being stored
func (t *transform) Transform(f *flow.Flow) interface{} {
	v2 := &recordV2{
		AccountID:   accountID,
		InterfaceID: strconv.FormatInt(f.Link.ID, 10),
		SrcAddr:     getNetworkA(f),
		DstAddr:     getNetworkB(f),
		SrcPort:     getTransportA(f),
		DstPort:     getTransportB(f),
		Protocol:    getProtocol(f),
		Packets:     int64(f.Metric.ABPackets + f.Metric.BAPackets),
		Bytes:       f.Metric.ABBytes + f.Metric.BABytes,
		Start:       int64(f.Start / 1000),
		End:         int64(f.Last / 1000),
		Action:      ActionAccept,
		LogStatus:   LogStatusOk,
	}

	v3 := &recordV3{
		recordV2:   *v2,
		VpcID:      "",
		SubnetID:   "",
		InstanceID: "",
		TCPFlags:   0, // TODO: extract from last packet
		Type:       getType(f),
		PktSrcAddr: "", // TODO: get outer flow via f.ParentUID
		PktDstAddr: "", // TODO: get outer flow via f.ParentUID
	}

	switch t.version {
	case 3:
		v3.Version = 3
		return v3
	default:
		v2.Version = 2
		return v2
	}
}
