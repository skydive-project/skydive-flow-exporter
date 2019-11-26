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

package mod

import (
	"fmt"
	"testing"
	"time"

	cache "github.com/pmylund/go-cache"
	"github.com/skydive-project/skydive/common"
	"github.com/skydive-project/skydive/flow"

	"github.com/skydive-project/skydive-flow-exporter/core"
)

type fakeResolver struct {
}

func (c *fakeResolver) IPToContext(ipString, nodeTID string) (*PeerContext, error) {
	switch ipString {
	case "192.168.0.5":
		return &PeerContext{
			Type: "pod",
			Name: "one-namespace/fake-pod-one",
			Set:  "one-namespace/replica-set-one",
		}, nil
	case "173.194.40.147":
		return &PeerContext{
			Type: "pod",
			Name: "two-namespace/fake-pod-two",
			Set:  "two-namespace/replica-set-two",
		}, nil
	default:
		return nil, fmt.Errorf("fake graph client")
	}
}

func (c *fakeResolver) TIDToType(nodeTID string) (string, error) {
	return "fake_node_type", nil
}

func getTestTransformer() core.Transformer {
	return &securityAdvisorFlowTransformer{
		flowUpdateCountCache: cache.New(10*time.Minute, 10*time.Minute),
		resolver:             &fakeResolver{},
		excludeStartedFlows:  false,
	}
}

func assertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Equal assertion failed: (expected: %T:%#v, actual: %T:%#v)", expected, expected, actual, actual)
	}
}

func assertEqualInt64(t *testing.T, expected, actual int64) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Equal assertion failed: (expected: %v, actual: %v)", expected, actual)
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
			A:        "192.168.0.5",
			B:        "173.194.40.147",
		},
		Transport: &flow.TransportLayer{
			Protocol: flow.FlowProtocol_TCP,
			A:        47838,
			B:        80,
		},
		Metric: &flow.FlowMetric{
			ABPackets: 6,
			ABBytes:   516,
			BAPackets: 4,
			BABytes:   760,
		},
		Start:   start,
		Last:    start,
		NodeTID: "probe-tid",
	}
}

func Test_Transform_basic_flow(t *testing.T) {
	transformer := getTestTransformer()
	f := getFlow()
	secAdvFlow := transformer.Transform(f).(*SecurityAdvisorFlow)
	assertEqual(t, version, secAdvFlow.Version)
	assertEqualInt64(t, 0, secAdvFlow.UpdateCount)
	assertEqual(t, "STARTED", secAdvFlow.Status)
	assertEqualInt64(t, 1546338030000, secAdvFlow.Start)
	assertEqualInt64(t, 1546338030000, secAdvFlow.Last)
	assertEqual(t, "fake_node_type", secAdvFlow.NodeType)
	// Test legacy container names
	assertEqual(t, "0_0_one-namespace/fake-pod-one_0", secAdvFlow.Network.AName)
	assertEqual(t, "0_0_two-namespace/fake-pod-two_0", secAdvFlow.Network.BName)
	// Test container context
	assertEqual(t, PeerTypePod, secAdvFlow.Context.A.Type)
	assertEqual(t, "one-namespace/fake-pod-one", secAdvFlow.Context.A.Name)
	assertEqual(t, "one-namespace/replica-set-one", secAdvFlow.Context.A.Set)
	assertEqual(t, PeerTypePod, secAdvFlow.Context.B.Type)
	assertEqual(t, "two-namespace/fake-pod-two", secAdvFlow.Context.B.Name)
	assertEqual(t, "two-namespace/replica-set-two", secAdvFlow.Context.B.Set)
	// Test that transport layer ports are encoded as strings
	assertEqual(t, "47838", secAdvFlow.Transport.A)
	assertEqual(t, "80", secAdvFlow.Transport.B)
}

func Test_Transform_UpdateCount_increses(t *testing.T) {
	transformer := getTestTransformer()
	f := getFlow()
	for i := int64(0); i < 10; i++ {
		secAdvFlow := transformer.Transform(f).(*SecurityAdvisorFlow)
		assertEqualInt64(t, i, secAdvFlow.UpdateCount)
	}
}

func Test_Transform_Status_updates(t *testing.T) {
	transformer := getTestTransformer()
	f := getFlow()
	secAdvFlow := transformer.Transform(f).(*SecurityAdvisorFlow)
	assertEqual(t, "STARTED", secAdvFlow.Status)
	secAdvFlow = transformer.Transform(f).(*SecurityAdvisorFlow)
	assertEqual(t, "UPDATED", secAdvFlow.Status)
	f.FinishType = flow.FlowFinishType_TCP_FIN
	secAdvFlow = transformer.Transform(f).(*SecurityAdvisorFlow)
	assertEqual(t, "ENDED", secAdvFlow.Status)
}

func getTestTransformerWithLocalTopology(t *testing.T) core.Transformer {
	localGremlinClient := newLocalGremlinQueryHelper(newKubernetesOnRuncTopologyGraph(t))

	runcResolver := &resolveRunc{localGremlinClient}
	dockerResolver := &resolveDocker{localGremlinClient}
	resolver := NewResolveMulti(runcResolver, dockerResolver)
	resolver = NewResolveFallback(resolver)
	resolver = NewResolveCache(resolver)

	return &securityAdvisorFlowTransformer{
		flowUpdateCountCache: cache.New(10*time.Minute, 10*time.Minute),
		resolver:             resolver,
		excludeStartedFlows:  false,
	}
}

func getRuncFlow() *flow.Flow {
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
			A:        "172.30.60.108",
			B:        "111.112.113.114",
		},
		Transport: &flow.TransportLayer{
			Protocol: flow.FlowProtocol_TCP,
			A:        47838,
			B:        80,
		},
		Metric: &flow.FlowMetric{
			ABPackets: 6,
			ABBytes:   516,
			BAPackets: 4,
			BABytes:   760,
		},
		Start:   start,
		Last:    start,
		NodeTID: "c3687053-ba82-5ccc-4c43-73a297f55f47",
	}
}

func TestTransformShouldResolveRuncContainerContext(t *testing.T) {
	transformer := getTestTransformerWithLocalTopology(t)
	f := getRuncFlow()
	secAdvFlow := transformer.Transform(f).(*SecurityAdvisorFlow)
	assertEqual(t, "netns", secAdvFlow.NodeType)
	assertEqual(t, "0_0_kube-system/kubernetes-dashboard-7996b848f4-pmv4z_0", secAdvFlow.Network.AName) // Legacy field
	assertEqual(t, PeerTypePod, secAdvFlow.Context.A.Type)
	assertEqual(t, "kube-system/kubernetes-dashboard-7996b848f4-pmv4z", secAdvFlow.Context.A.Name)
	assertEqual(t, "ReplicaSet:kube-system/kubernetes-dashboard-7996b848f4", secAdvFlow.Context.A.Set)

	// Name should be blank for IPs that are not pods
	assertEqual(t, "", secAdvFlow.Network.BName)
	if secAdvFlow.Context.B != nil {
		t.Fatal("Expected Context.B to be nil")
	}
}
