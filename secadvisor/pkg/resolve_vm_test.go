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

func newTestResolveVM(t *testing.T) Resolver {
	return &resolveVM{gremlinClient: newLocalGremlinQueryHelper(newVMTopologyGraph(t))}
}

func TestResolveVMShouldFindHostContext(t *testing.T) {
	r := newTestResolveVM(t)
	expected := &PeerContext{
		Type: "host",
		Name: "my-host-name-1",
	}
	actual, err := r.IPToContext("100.101.102.103", "09dcdca2-4259-5df9-47fc-e4bed4eac0ed")
	if err != nil {
		t.Fatalf("IPToContext failed: %v", err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected: %+v, got: %+v", expected, actual)
	}
}

func TestResolveVMShouldNotFindContextOfNonExistingIP(t *testing.T) {
	r := newTestResolveVM(t)
	actual, err := r.IPToContext("11.22.33.44", "09dcdca2-4259-5df9-47fc-e4bed4eac0ed")
	if err == nil {
		t.Errorf("Expected error but got none")
	}
	if actual != nil {
		t.Errorf("Expected empty response, got: %v", actual)
	}
}

func TestResolveVMShouldFindTIDType(t *testing.T) {
	r := newTestResolveVM(t)
	expected := "device"
	actual, err := r.TIDToType("09dcdca2-4259-5df9-47fc-e4bed4eac0ed")
	if err != nil {
		t.Fatalf("TIDToType failed: %v", err)
	}
	if actual != expected {
		t.Errorf("Expected: %v, got: %v", expected, actual)
	}
}

func TestResolveVMShouldNotFindTIDTypeOfNonExistingTID(t *testing.T) {
	r := newTestResolveVM(t)
	actual, err := r.TIDToType("11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Error("Expected error but got none")
	}
	if actual != "" {
		t.Errorf("Expected empty response, got: %v", actual)
	}
}
