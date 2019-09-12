package mod

import (
	"reflect"
	"testing"
)

func newTestResolveDocker(t *testing.T) Resolver {
	return &resolveDocker{gremlinClient: newLocalGremlinQueryHelper(newDockerTopologyGraph(t))}
}

func newTestResolveKubernetesOnDocker(t *testing.T) Resolver {
	return &resolveDocker{gremlinClient: newLocalGremlinQueryHelper(newKubernetesOnDockerTopologyGraph(t))}
}

func TestResolveDockerShouldFindContainerContext(t *testing.T) {
	r := newTestResolveDocker(t)
	expected := &PeerContext{
		Type: "container",
		Name: "pinger-container-1",
	}
	nameTIDs := []string{
		"eac0f98c-2ab0-5b89-6490-9e8816f8cba3", // eth0 interface inside the container netns
		"38e2f253-2305-5e91-5af5-2bfcab208b1a", // veth interface
		"577f878a-1e7f-5b2d-60f0-efc7ff5da510", // docker0 bridge
	}
	for _, nameTID := range nameTIDs {
		actual, err := r.IPToContext("172.17.0.3", nameTID)
		if err != nil {
			t.Fatalf("IPToContext (nameTID=%s) failed: %v", nameTID, err)
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("Expected: %+v, got: %+v", expected, actual)
		}
	}
}

func TestResolveDockerShouldNotFindContextOfNonExistingIP(t *testing.T) {
	r := newTestResolveDocker(t)
	actual, err := r.IPToContext("8.7.6.5", "eac0f98c-2ab0-5b89-6490-9e8816f8cba3")
	if err == nil {
		t.Errorf("Expected error but got none")
	}
	if actual != nil {
		t.Errorf("Expected empty response, got: %v", actual)
	}
}

func TestResolveDockerShouldFindTIDType(t *testing.T) {
	r := newTestResolveDocker(t)
	expected := "bridge"
	actual, err := r.TIDToType("577f878a-1e7f-5b2d-60f0-efc7ff5da510")
	if err != nil {
		t.Fatalf("TIDToType failed: %v", err)
	}
	if actual != expected {
		t.Errorf("Expected: %v, got: %v", expected, actual)
	}
}

func TestResolveDockerShouldNotFindTIDTypeOfNonExistingTID(t *testing.T) {
	r := newTestResolveDocker(t)
	actual, err := r.TIDToType("11111111-1111-1111-1111-111111111111")
	if err == nil {
		t.Error("Expected error but got none")
	}
	if actual != "" {
		t.Errorf("Expected empty response, got: %v", actual)
	}
}

func TestResolveKubernetesOnDockerShouldFindPodInfo(t *testing.T) {
	r := newTestResolveKubernetesOnDocker(t)
	expected := &PeerContext{
		Type: "pod",
		Name: "default/pinger-depl-867fbd4567-8fdwd",
		Set:  "ReplicaSet:default/pinger-depl-867fbd4567",
	}
	actual, err := r.IPToContext("172.17.0.5", "460e53ed-2cc4-5116-69b0-f5fe754a31b2")
	if err != nil {
		t.Fatalf("IPToContext failed: %v", err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected: %+v, got: %+v", expected, actual)
	}
}
