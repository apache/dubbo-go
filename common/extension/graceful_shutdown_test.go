package extension

import (
	"context"
	"testing"
)

func TestGetAllGracefulShutdownCallbacksReturnsSnapshot(t *testing.T) {
	t.Cleanup(func() {
		gracefulShutdownCallbacksMu.Lock()
		delete(gracefulShutdownCallbacks, "snapshot-test")
		gracefulShutdownCallbacksMu.Unlock()
	})

	SetGracefulShutdownCallback("snapshot-test", func(context.Context) error {
		return nil
	})

	callbacks := GetAllGracefulShutdownCallbacks()
	delete(callbacks, "snapshot-test")

	if _, ok := GetGracefulShutdownCallback("snapshot-test"); !ok {
		t.Fatal("expected stored callback to remain after mutating returned snapshot")
	}
}
