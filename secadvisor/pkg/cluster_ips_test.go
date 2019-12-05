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
	"reflect"
	"sort"
	"testing"

	"github.com/skydive-project/skydive/graffiti/graph"
)

func newKubernetesNodesWithExternalIPsTopologyGraph(t *testing.T) *graph.Graph {
	g := newGraph(t)
	g.NewNode(graph.GenID(), graph.Metadata{
		"Manager": "k8s",
		"Type":    "node",
		"Name":    "10.74.144.91",
		"K8s": map[string]interface{}{
			"Extra": map[string]interface{}{
				"Status": map[string]interface{}{
					"Addresses": []interface{}{
						map[string]interface{}{
							"Address": "10.74.144.91",
							"Type":    "InternalIP",
						},
						map[string]interface{}{
							"Address": "169.63.32.36",
							"Type":    "ExternalIP",
						},
						map[string]interface{}{
							"Address": "10.74.144.91",
							"Type":    "Hostname",
						},
					},
				},
			},
		},
	})
	g.NewNode(graph.GenID(), graph.Metadata{
		"Manager": "k8s",
		"Type":    "node",
		"Name":    "10.74.144.75",
		"K8s": map[string]interface{}{
			"Extra": map[string]interface{}{
				"Status": map[string]interface{}{
					"Addresses": []interface{}{
						map[string]interface{}{
							"Address": "10.74.144.75",
							"Type":    "InternalIP",
						},
						map[string]interface{}{
							"Address": "169.63.32.43",
							"Type":    "ExternalIP",
						},
						map[string]interface{}{
							"Address": "10.74.144.75",
							"Type":    "Hostname",
						},
					},
				},
			},
		},
	})
	g.NewNode(graph.GenID(), graph.Metadata{
		"Manager": "k8s",
		"Type":    "node",
		"Name":    "10.74.144.77",
		"K8s": map[string]interface{}{
			"Extra": map[string]interface{}{
				"Status": map[string]interface{}{
					"Addresses": []interface{}{
						map[string]interface{}{
							"Address": "10.74.144.77",
							"Type":    "InternalIP",
						},
						map[string]interface{}{
							"Address": "169.63.32.45",
							"Type":    "ExternalIP",
						},
						map[string]interface{}{
							"Address": "10.74.144.77",
							"Type":    "Hostname",
						},
					},
				},
			},
		},
	})
	return g
}

func newKubernetesNodesWithoutExternalIPsTopologyGraph(t *testing.T) *graph.Graph {
	g := newGraph(t)
	g.NewNode(graph.GenID(), graph.Metadata{
		"Manager": "k8s",
		"Type":    "node",
		"Name":    "10.74.144.91",
		"K8s": map[string]interface{}{
			"Extra": map[string]interface{}{
				"Status": map[string]interface{}{
					"Addresses": []interface{}{
						map[string]interface{}{
							"Address": "10.74.144.91",
							"Type":    "InternalIP",
						},
						map[string]interface{}{
							"Address": "10.74.144.91",
							"Type":    "Hostname",
						},
					},
				},
			},
		},
	})
	g.NewNode(graph.GenID(), graph.Metadata{
		"Manager": "k8s",
		"Type":    "node",
		"Name":    "10.74.144.75",
		"K8s": map[string]interface{}{
			"Extra": map[string]interface{}{
				"Status": map[string]interface{}{
					"Addresses": []interface{}{
						map[string]interface{}{
							"Address": "10.74.144.75",
							"Type":    "InternalIP",
						},
						map[string]interface{}{
							"Address": "10.74.144.75",
							"Type":    "Hostname",
						},
					},
				},
			},
		},
	})
	return g
}

func TestFindClusterNodesIpsShouldFindExternalIPs(t *testing.T) {
	gremlinClient := newLocalGremlinQueryHelper(newKubernetesNodesWithExternalIPsTopologyGraph(t))
	ips, err := FindClusterNodesIPs(gremlinClient)
	if err != nil {
		t.Fatalf("findClusterNodesIPs failed: %v", err)
	}
	sort.Sort(sort.StringSlice(ips))
	expected := []string{"169.63.32.36", "169.63.32.43", "169.63.32.45"}
	if !reflect.DeepEqual(expected, ips) {
		t.Errorf("Expected: %+v, got: %+v", expected, ips)
	}
}

func TestFindClusterNodesIpsShouldFindInternalIPsWhenThereAreNoExternalIPs(t *testing.T) {
	gremlinClient := newLocalGremlinQueryHelper(newKubernetesNodesWithoutExternalIPsTopologyGraph(t))
	ips, err := FindClusterNodesIPs(gremlinClient)
	if err != nil {
		t.Fatalf("findClusterNodesIPs failed: %v", err)
	}
	sort.Sort(sort.StringSlice(ips))
	expected := []string{"10.74.144.75", "10.74.144.91"}
	if !reflect.DeepEqual(expected, ips) {
		t.Errorf("Expected: %+v, got: %+v", expected, ips)
	}
}

func TestConvertIPsToNetmasks(t *testing.T) {
	netmasks := ConvertIPsToNetmasks([]string{"1.2.3.4", "5.6.7.8"})
	expected := []string{"1.2.3.4/32", "5.6.7.8/32"}
	if !reflect.DeepEqual(expected, netmasks) {
		t.Errorf("Expected: %+v, got: %+v", expected, netmasks)
	}
}
