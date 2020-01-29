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
	"github.com/skydive-project/skydive/graffiti/graph"
	g "github.com/skydive-project/skydive/gremlin"
)

// ConvertIPsToNetmasks converts a list of IPs to a list of single-host
// netmasks which can be used for cluster netmasks.
func ConvertIPsToNetmasks(ips []string) []string {
	netmasks := []string{}
	for _, ip := range ips {
		netmasks = append(netmasks, ip+"/32")
	}
	return netmasks
}

// FindClusterNodesIPs returns the list of IP addresses of the Kubernetes
// cluster nodes. External IPs are preferred, but if not found then internal
// IPs are returned.
func FindClusterNodesIPs(gremlinClient GremlinNodeGetter) ([]string, error) {
	nodes, err := gremlinClient.GetNodes(g.G.V().Has("Manager", "k8s", "Type", "node"))
	if err != nil {
		return []string{}, err
	}

	ips := []string{}
	for _, node := range nodes {
		ip, err := extractNodeIP(node)
		if err == nil && ip != "" {
			ips = append(ips, ip)
		}
	}
	return ips, nil
}

// Extract the first ExternalIP found, or, if none exists, the InternalIP
// found; otherwise returns an empty string
func extractNodeIP(node *graph.Node) (string, error) {
	addressesObj, err := node.GetField("K8s.Extra.Status.Addresses")
	if err != nil {
		return "", err
	}

	addresses, ok := addressesObj.([]interface{})
	if !ok {
		return "", err
	}

	internalIP := ""
	for _, addressObj := range addresses {
		address, ok := addressObj.(map[string]interface{})
		if !ok {
			continue
		}
		addrType, ok := address["Type"].(string)
		if !ok {
			continue
		}
		ip, ok := address["Address"].(string)
		if !ok {
			continue
		}
		switch addrType {
		case "ExternalIP":
			return ip, nil
		case "InternalIP":
			internalIP = ip
		}
	}
	return internalIP, nil
}
