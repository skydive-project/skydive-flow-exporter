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
	"github.com/skydive-project/skydive/graffiti/graph"
	g "github.com/skydive-project/skydive/gremlin"
)

func queryNodeType(gremlinClient GremlinNodeGetter, nodeTID string) (string, error) {
	node, err := gremlinClient.GetNode(g.G.V().Has("TID", nodeTID))
	if err != nil {
		return "", err
	}

	return node.GetFieldString("Type")
}

func queryPodByContainerID(gremlinClient GremlinNodeGetter,
	containerIDPrefix, containerID string) (*PeerContext, error) {
	containerIDValue := containerIDPrefix + "://" + containerID
	node, err := gremlinClient.GetNode(g.G.V().Has("K8s.Extra.Status.ContainerStatuses.ContainerID", containerIDValue))
	if err != nil {
		return nil, err
	}

	namespace, err := node.GetFieldString("K8s.Namespace")
	if err != nil {
		return nil, err
	}
	podName, err := node.GetFieldString("K8s.Name")
	if err != nil {
		return nil, err
	}
	name := namespace + "/" + podName

	set := extractSet(node, namespace)

	return &PeerContext{
		Type: PeerTypePod,
		Name: name,
		Set:  set,
	}, nil
}

func extractSet(node *graph.Node, namespace string) string {
	ownerRefs, err := node.GetField("K8s.Extra.ObjectMeta.OwnerReferences")
	if err != nil {
		return ""
	}
	ownerRefsList, ok := ownerRefs.([]interface{})
	if !ok || len(ownerRefsList) < 1 {
		return ""
	}
	ownerRef, ok := ownerRefsList[0].(map[string]interface{})
	if !ok {
		return ""
	}
	// kind can be "ReplicaSet" or "DaemonSet"
	kind, ok := ownerRef["Kind"].(string)
	if !ok {
		return ""
	}
	name, ok := ownerRef["Name"].(string)
	if !ok {
		return ""
	}
	return kind + ":" + namespace + "/" + name
}
