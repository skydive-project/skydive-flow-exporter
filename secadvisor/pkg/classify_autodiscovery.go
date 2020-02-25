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
	"time"

	"github.com/spf13/viper"

	"github.com/skydive-project/skydive-flow-exporter/core"
	"github.com/skydive-project/skydive/api/client"
	"github.com/skydive-project/skydive/logging"
)

// NewClassifySubnetWithAutoDiscovery creates a ClassifySubnet object with
// additional discovery of Kubernetes cluster nodes IP netmasks.
func NewClassifySubnetWithAutoDiscovery(cfg *viper.Viper) (interface{}, error) {
	clusterNetMasks := cfg.GetStringSlice(core.CfgRoot + "classify.cluster_net_masks")
	gremlinClient := client.NewGremlinQueryHelper(core.CfgAuthOpts(cfg))
	nodesIPs := getClusterNodesIPsWithRetries(gremlinClient)
	nodesNetmasks := ConvertIPsToNetmasks(nodesIPs)
	logging.GetLogger().Infof("Adding Kubenetes cluster nodes net masks: %v", nodesNetmasks)
	clusterNetMasks = append(clusterNetMasks, nodesNetmasks...)
	return core.NewClassifySubnetFromList(clusterNetMasks)
}

func getClusterNodesIPsWithRetries(gremlinClient GremlinNodeGetter) []string {
	logging.GetLogger().Infof("Querying Skydive analzyer topology for Kubenetes cluster nodes information")
	for {
		nodesIPs, err := FindClusterNodesIPs(gremlinClient)
		if err == nil {
			return nodesIPs
		} else {
			logging.GetLogger().Errorf("Failed querying skydive analyzer (retrying): %v", err)
			time.Sleep(1 * time.Second)
		}
	}
	return []string{} // never reached
}
