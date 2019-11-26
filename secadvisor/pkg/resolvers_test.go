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
	"strings"
	"testing"

	"github.com/skydive-project/skydive/common"
	"github.com/skydive-project/skydive/graffiti/graph"
	"github.com/skydive-project/skydive/graffiti/graph/traversal"
	"github.com/skydive-project/skydive/gremlin"
)

type localGremlinQueryHelper struct {
	graph *graph.Graph
}

func (l *localGremlinQueryHelper) Request(query interface{}) ([]interface{}, error) {
	queryString := gremlin.NewQueryStringFromArgument(query).String()
	ts, err := traversal.NewGremlinTraversalParser().Parse(strings.NewReader(queryString))
	if err != nil {
		return nil, err
	}

	res, err := ts.Exec(l.graph, false)
	if err != nil {
		return nil, err
	}

	return res.Values(), nil
}

func (l *localGremlinQueryHelper) GetNodes(query interface{}) ([]*graph.Node, error) {
	result, err := l.Request(query)
	if err != nil {
		return nil, err
	}
	nodes := make([]*graph.Node, 0, len(result))
	for _, item := range result {
		switch item.(type) {
		case *graph.Node:
			nodes = append(nodes, item.(*graph.Node))
		case []*graph.Node:
			for _, i := range item.([]*graph.Node) {
				nodes = append(nodes, i)
			}
		default:
			return nil, fmt.Errorf("Unknown type %T of item", item)
		}
	}
	return nodes, nil
}

func (l *localGremlinQueryHelper) GetNode(query interface{}) (*graph.Node, error) {
	nodes, err := l.GetNodes(query)
	if err != nil {
		return nil, err
	}

	if len(nodes) > 0 {
		return nodes[0], nil
	}

	return nil, common.ErrNotFound
}

func newLocalGremlinQueryHelper(graph *graph.Graph) *localGremlinQueryHelper {
	return &localGremlinQueryHelper{graph}
}

func newGraph(t *testing.T) *graph.Graph {
	b, err := graph.NewMemoryBackend()
	if err != nil {
		t.Error(err)
	}
	return graph.NewGraph("testhost", b, common.UnknownService)
}

func newRuncTopologyGraph(t *testing.T) *graph.Graph {
	g := newGraph(t)
	n1, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Manager": "runc",
		"Type":    "netns",
		"TID":     "ce2ed4fb-1340-57b1-796f-5d648665aed7",
	})
	n2, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Manager": "runc",
		"Type":    "container",
		"Runc": map[string]interface{}{
			"ContainerID": "82cd545921e50c50aab4166c92d14f91e47e2bf59ebb07e784ffd60609438609",
			"Hosts": map[string]interface{}{
				"IP":       "172.30.149.34",
				"Hostname": "my-container-name-5bbc557665-h66vq",
			},
		},
	})
	g.Link(n1, n2, graph.Metadata{})
	return g
}

func newDockerTopologyGraph(t *testing.T) *graph.Graph {
	g := newGraph(t)

	hostNode, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type": "host",
		"Name": "dummy-host",
		"TID":  "3ac60fae-bf77-5a60-548f-21d5663ffdeb",
	})
	eth0Node, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type":   "device",
		"Driver": "vif",
		"IPV4": []string{
			"155.166.177.188/23",
			"111.122.133.144/23",
		},
		"Name": "eth0",
		"TID":  "09dcdca2-4259-5df9-47fc-e4bed4eac0ed",
	})
	bridgeNode, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type": "bridge",
		"IPV4": []string{"172.21.0.1/16"},
		"Name": "br-9254261aa549",
		"TID":  "b2772b02-934a-5ffa-4934-2a9ffbc4abc0",
	})
	docker0Node, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type": "bridge",
		"IPV4": []string{"172.21.0.1/16"},
		"Name": "docker0",
		"TID":  "577f878a-1e7f-5b2d-60f0-efc7ff5da510",
	})
	vethNode, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type":   "veth",
		"Driver": "veth",
		"Name":   "vethda954f5",
		"TID":    "38e2f253-2305-5e91-5af5-2bfcab208b1a",
	})
	netnsNode, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type":    "netns",
		"Manager": "docker",
		"Name":    "62c9732f61fe",
		"TID":     "39ad4916-7469-51e1-70f5-c8755262793e",
	})
	dockereth0Node, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type": "veth",
		"IPV4": []string{"172.17.0.3/16"},
		"Name": "eth0",
		"TID":  "eac0f98c-2ab0-5b89-6490-9e8816f8cba3",
	})
	containerNode, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type":    "container",
		"Name":    "pinger-container-1",
		"Manager": "docker",
		"Docker": map[string]interface{}{
			"ContainerID":   "3ee3a7a6e45bf2fc4bdb213cc348b317db7f6d30c285b1fa621c4bde1b0ca3ac",
			"ContainerName": "pinger-container-1",
		},
		"TID": "4ac94353-9719-5342-7f34-e12074d37402",
	})

	g.Link(hostNode, netnsNode, graph.Metadata{})
	g.Link(hostNode, eth0Node, graph.Metadata{})
	g.Link(hostNode, bridgeNode, graph.Metadata{})
	g.Link(hostNode, docker0Node, graph.Metadata{})
	g.Link(hostNode, vethNode, graph.Metadata{})
	g.Link(docker0Node, vethNode, graph.Metadata{})
	g.Link(vethNode, dockereth0Node, graph.Metadata{})
	g.Link(netnsNode, dockereth0Node, graph.Metadata{})
	g.Link(netnsNode, containerNode, graph.Metadata{})

	return g
}

func newKubernetesOnRuncTopologyGraph(t *testing.T) *graph.Graph {
	g := newGraph(t)

	netnsNode, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type":    "netns",
		"Manager": "runc",
		"Name":    "3709cd6138c28d5b56ed85d8aa05de7db40ba729c09e1df96c99d4e0d4cb0203",
		"Runtime": "runc",
		"TID":     "c3687053-ba82-5ccc-4c43-73a297f55f47",
	})
	containerNode, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type":    "container",
		"Runtime": "runc",
		"Manager": "runc",
		"Name":    "8a32bc39ff9c6d8555fdc0fa3af7b61c14fd7ccd3865193a8b07ce7087ec106b",
		"Runc": map[string]interface{}{
			"ContainerID": "8a32bc39ff9c6d8555fdc0fa3af7b61c14fd7ccd3865193a8b07ce7087ec106b",
			"Hosts": map[string]interface{}{
				"IP":       "172.30.60.108",
				"Hostname": "kubernetes-dashboard-7996b848f4-pmv4z",
			},
		},
		"TID": "7763a2ac-77ea-5e8f-475d-3de12b6c740d",
	})
	g.NewNode(graph.GenID(), graph.Metadata{
		"Type":    "pod",
		"Manager": "k8s",
		"Name":    "kubernetes-dashboard-7996b848f4-pmv4z",
		"K8s": map[string]interface{}{
			"Extra": map[string]interface{}{
				"ObjectMeta": map[string]interface{}{
					"OwnerReferences": []interface{}{
						map[string]interface{}{
							"Kind": "ReplicaSet",
							"Name": "kubernetes-dashboard-7996b848f4",
						},
					},
				},
				"Status": map[string]interface{}{
					"ContainerStatuses": []interface{}{
						map[string]interface{}{
							"ContainerID": "containerd://8a32bc39ff9c6d8555fdc0fa3af7b61c14fd7ccd3865193a8b07ce7087ec106b",
						},
					},
				},
			},
			"IP":        "172.30.60.108",
			"Name":      "kubernetes-dashboard-7996b848f4-pmv4z",
			"Namespace": "kube-system",
		},
	})

	g.Link(netnsNode, containerNode, graph.Metadata{})

	return g
}

func newKubernetesOnDockerTopologyGraph(t *testing.T) *graph.Graph {
	g := newGraph(t)

	dockereth0Node, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type": "veth",
		"IPV4": []string{"172.17.0.5/16"},
		"Name": "eth0",
		"TID":  "460e53ed-2cc4-5116-69b0-f5fe754a31b2",
	})
	netnsNode, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type":    "netns",
		"Manager": "docker",
		"Name":    "k8s_pinger-two_pinger-depl-867fbd4567-8fdwd_default_38b64484-ddd2-11e9-9c32-06615272ae54_0",
		"TID":     "d4e62829-bfe5-5a0d-5676-8459f8428ac4",
	})
	containerNode, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type":    "container",
		"Name":    "k8s_pinger-two_pinger-depl-867fbd4567-8fdwd_default_38b64484-ddd2-11e9-9c32-06615272ae54_0",
		"Manager": "docker",
		"Docker": map[string]interface{}{
			"ContainerID":   "c8be05f0616091df905d8aa409431ae4061e9af2881c6bc6ee3abb19b7aa1eb9",
			"ContainerName": "k8s_pinger-two_pinger-depl-867fbd4567-8fdwd_default_38b64484-ddd2-11e9-9c32-06615272ae54_0",
		},
		"TID": "1062e3fa-927f-55bb-4b9d-b22be94fd22b",
	})
	g.NewNode(graph.GenID(), graph.Metadata{
		"Type":    "pod",
		"Manager": "k8s",
		"Name":    "pinger-depl-867fbd4567-8fdwd",
		"K8s": map[string]interface{}{
			"Extra": map[string]interface{}{
				"ObjectMeta": map[string]interface{}{
					"OwnerReferences": []interface{}{
						map[string]interface{}{
							"Kind": "ReplicaSet",
							"Name": "pinger-depl-867fbd4567",
						},
					},
				},
				"Status": map[string]interface{}{
					"ContainerStatuses": []interface{}{
						map[string]interface{}{
							"ContainerID": "docker://28ee7186dc2ff973337680cdd16d987f010e34d56e236e70a465481b6693e05e",
						},
						map[string]interface{}{
							"ContainerID": "docker://c8be05f0616091df905d8aa409431ae4061e9af2881c6bc6ee3abb19b7aa1eb9",
						},
					},
				},
			},
			"IP":        "172.17.0.5",
			"Name":      "pinger-depl-867fbd4567-8fdwd",
			"Namespace": "default",
		},
	})

	g.Link(netnsNode, dockereth0Node, graph.Metadata{})
	g.Link(netnsNode, containerNode, graph.Metadata{})

	return g
}

func newVMTopologyGraph(t *testing.T) *graph.Graph {
	g := newGraph(t)
	hostNode, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type": "host",
		"Name": "my-host-name-1",
		"TID":  "3ac60fae-bf77-5a60-548f-21d5663ffdeb",
	})
	eth0Node, _ := g.NewNode(graph.GenID(), graph.Metadata{
		"Type": "device",
		"TID":  "09dcdca2-4259-5df9-47fc-e4bed4eac0ed",
		"RoutingTables": []interface{}{
			map[string]interface{}{
				"ID":  254,
				"Src": nil,
			},
			map[string]interface{}{
				"ID":  254,
				"Src": "100.101.102.103",
			},
		},
	})
	g.Link(hostNode, eth0Node, graph.Metadata{})
	return g
}
