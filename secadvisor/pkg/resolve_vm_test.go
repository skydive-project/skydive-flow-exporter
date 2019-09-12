package mod

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
