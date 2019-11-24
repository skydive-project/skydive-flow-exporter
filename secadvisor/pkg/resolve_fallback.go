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

type resolveFallback struct {
	resolver Resolver
}

// NewResolveFallback creates a new name resolver
func NewResolveFallback(resolver Resolver) Resolver {
	return &resolveFallback{
		resolver: resolver,
	}
}

// IPToContext resolves IP address to Peer context
func (rc *resolveFallback) IPToContext(ipString, nodeTID string) (*PeerContext, error) {
	if ipString == "" {
		return nil, nil
	}

	context, err := rc.resolver.IPToContext(ipString, nodeTID)
	if err != nil {
		return nil, nil
	}

	return context, nil
}

// TIDToType resolve tid to type
func (rc *resolveFallback) TIDToType(nodeTID string) (string, error) {
	if nodeTID == "" {
		return "", nil
	}

	nodeType, _ := rc.resolver.TIDToType(nodeTID)
	return nodeType, nil
}
