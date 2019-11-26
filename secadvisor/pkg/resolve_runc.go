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
	"github.com/skydive-project/skydive/api/client"
	"github.com/skydive-project/skydive/graffiti/graph"
	g "github.com/skydive-project/skydive/gremlin"
	"github.com/spf13/viper"

	"github.com/skydive-project/skydive-flow-exporter/core"
)

// GremlinNodeGetter interface allows access to get topology nodes according to
// a gremlin query.
type GremlinNodeGetter interface {
	GetNodes(query interface{}) ([]*graph.Node, error)
	GetNode(query interface{}) (*graph.Node, error)
}

// NewResolveRunc creates a new name resolver
func NewResolveRunc(cfg *viper.Viper) Resolver {
	gremlinClient := client.NewGremlinQueryHelper(core.CfgAuthOpts(cfg))
	return &resolveRunc{
		gremlinClient: gremlinClient,
	}
}

type resolveRunc struct {
	gremlinClient GremlinNodeGetter
}

// IPToContext resolves IP address to Peer context
func (r *resolveRunc) IPToContext(ipString, nodeTID string) (*PeerContext, error) {
	node, err := r.gremlinClient.GetNode(g.G.V().Has("Runc.Hosts.IP", ipString).ShortestPathTo(g.Metadata("TID", nodeTID)))
	if err != nil {
		return nil, err
	}

	name, err := node.GetFieldString("Runc.Hosts.Hostname")
	if err != nil {
		name = ""
	}

	containerID, err := node.GetFieldString("Runc.ContainerID")
	if err != nil {
		containerID = ""
	}

	podPeerContext, err := queryPodByContainerID(r.gremlinClient, "containerd", containerID)
	if err != nil {
		// No pod information
		return &PeerContext{
			Type: PeerTypeContainer,
			Name: name,
		}, nil
	}

	return podPeerContext, nil
}

// TIDToType resolve tid to type
func (r *resolveRunc) TIDToType(nodeTID string) (string, error) {
	return queryNodeType(r.gremlinClient, nodeTID)
}
