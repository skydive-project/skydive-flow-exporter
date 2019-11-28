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
	"testing"
)

func newTestResolveRunc(t *testing.T) Resolver {
	return &resolveRunc{gremlinClient: newLocalGremlinQueryHelper(newRuncTopologyGraph(t))}
}

func TestResolveRuncShouldFindContainerContext(t *testing.T) {
	r := newTestResolveRunc(t)
	expected := &PeerContext{
		Type: "container",
		Name: "my-container-name-5bbc557665-h66vq",
	}
	actual, err := r.IPToContext("172.30.149.34", "ce2ed4fb-1340-57b1-796f-5d648665aed7")
	if err != nil {
		t.Fatalf("IPToContext failed: %v", err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected: %+v, got: %+v", expected, actual)
	}
}

func TestResolveRuncShouldNotFindContextOfNonExistingIP(t *testing.T) {
	r := newTestResolveRunc(t)
	actual, err := r.IPToContext("11.22.33.44", "ce2ed4fb-1340-57b1-796f-5d648665aed7")
	if err == nil {
		t.Errorf("Expected error but got none")
	}
	if actual != nil {
		t.Errorf("Expected empty response, got: %v", actual)
	}
}

func TestResolveRuncShouldFindTIDType(t *testing.T) {
	r := newTestResolveRunc(t)
	expected := "netns"
	actual, err := r.TIDToType("ce2ed4fb-1340-57b1-796f-5d648665aed7")
	if err != nil {
		t.Fatalf("TIDToType failed: %v", err)
	}
	if actual != expected {
		t.Errorf("Expected: %v, got: %v", expected, actual)
	}
}

func TestResolveRuncShouldNotFindTIDTypeOfNonExistingTID(t *testing.T) {
	r := newTestResolveRunc(t)
	actual, err := r.TIDToType("11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Error("Expected error but got none")
	}
	if actual != "" {
		t.Errorf("Expected empty response, got: %v", actual)
	}
}
