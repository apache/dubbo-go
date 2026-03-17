package extension

import (
	"container/list"
	"context"
	"testing"
)

func TestGracefulShutdownCallbacksReturnsSnapshot(t *testing.T) {
	t.Cleanup(func() {
		UnregisterGracefulShutdownCallback("snapshot-test")
	})

	RegisterGracefulShutdownCallback("snapshot-test", func(context.Context) error {
		return nil
	})

	callbacks := GracefulShutdownCallbacks()
	delete(callbacks, "snapshot-test")

	if _, ok := LookupGracefulShutdownCallback("snapshot-test"); !ok {
		t.Fatal("expected stored callback to remain after mutating returned snapshot")
	}
}

func TestGetAllCustomShutdownCallbacksReturnsSnapshot(t *testing.T) {
	customShutdownCallbacksMu.Lock()
	original := customShutdownCallbacks
	customShutdownCallbacks = list.New()
	customShutdownCallbacksMu.Unlock()

	t.Cleanup(func() {
		customShutdownCallbacksMu.Lock()
		customShutdownCallbacks = original
		customShutdownCallbacksMu.Unlock()
	})

	AddCustomShutdownCallback(func() {})
	callbacks := GetAllCustomShutdownCallbacks()
	callbacks.PushBack(func() {})

	if got := GetAllCustomShutdownCallbacks().Len(); got != 1 {
		t.Fatalf("expected custom callback snapshot mutation not to affect stored callbacks, got %d", got)
	}
}
