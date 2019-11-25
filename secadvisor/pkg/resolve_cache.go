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
	"time"

	cache "github.com/pmylund/go-cache"

	"github.com/skydive-project/skydive/common"
	"github.com/skydive-project/skydive/logging"
)

// Resolver resolves values for the transformer
type Resolver interface {
	IPToContext(ipString, nodeTID string) (*PeerContext, error)
	TIDToType(nodeTID string) (string, error)
}

type resolveCache struct {
	resolver     Resolver
	contextCache *cache.Cache
	typeCache    *cache.Cache
}

// NewResolveCache creates a new name resolver
func NewResolveCache(resolver Resolver) Resolver {
	return &resolveCache{
		resolver:     resolver,
		contextCache: cache.New(5*time.Minute, 10*time.Minute),
		typeCache:    cache.New(5*time.Minute, 10*time.Minute),
	}
}

// IPToContext resolves IP address to Peer context
func (rc *resolveCache) IPToContext(ipString, nodeTID string) (*PeerContext, error) {
	if ipString == "" {
		return nil, nil
	}

	cacheKey := ipString + "," + nodeTID
	cachedContext, ok := rc.contextCache.Get(cacheKey)
	if ok {
		return cachedContext.(*PeerContext), nil
	}

	context, err := rc.resolver.IPToContext(ipString, nodeTID)
	switch err {
	case nil:
		rc.contextCache.Set(cacheKey, context, cache.DefaultExpiration)
		return context, nil
	case common.ErrNotFound:
		rc.contextCache.Set(cacheKey, nil, cache.DefaultExpiration)
		return nil, nil
	default:
		logging.GetLogger().Warningf("Failed to query container context for IP '%s' in nodeTID '%s': %s",
			ipString, nodeTID, err)
		return nil, err
	}
}

// TIDToType resolve tid to type
func (rc *resolveCache) TIDToType(nodeTID string) (string, error) {
	nodeType, ok := rc.typeCache.Get(nodeTID)
	if !ok {
		var err error
		nodeType, err = rc.resolver.TIDToType(nodeTID)

		if err != nil {
			if err != common.ErrNotFound {
				logging.GetLogger().Warningf("Failed to query node type for TID '%s': %s", nodeTID, err)
			}
			return "", err
		}

		rc.typeCache.Set(nodeTID, nodeType, cache.DefaultExpiration)
	}

	return nodeType.(string), nil
}
