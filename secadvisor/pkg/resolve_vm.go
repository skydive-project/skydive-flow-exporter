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
	g "github.com/skydive-project/skydive/gremlin"
	"github.com/spf13/viper"

	"github.com/skydive-project/skydive-flow-exporter/core"
)

// NewResolveVM creates a new name resolver
func NewResolveVM(cfg *viper.Viper) Resolver {
	gremlinClient := client.NewGremlinQueryHelper(core.CfgAuthOpts(cfg))

	return &resolveVM{
		gremlinClient: gremlinClient,
	}
}

type resolveVM struct {
	gremlinClient GremlinNodeGetter
}

// IPToContext resolves IP address to Peer context
func (r *resolveVM) IPToContext(ipString, nodeTID string) (*PeerContext, error) {
	node, err := r.gremlinClient.GetNode(g.G.V().Has("RoutingTables.Src", ipString).In().Has("Type", "host"))
	if err != nil {
		return nil, err
	}

	name, err := node.GetFieldString("Name")
	if err != nil {
		return nil, err
	}

	return &PeerContext{
		Type: PeerTypeHost,
		Name: name,
	}, nil
}

// TIDToType resolve tid to type
func (r *resolveVM) TIDToType(nodeTID string) (string, error) {
	return queryNodeType(r.gremlinClient, nodeTID)
}
