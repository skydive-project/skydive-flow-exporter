package mod

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
