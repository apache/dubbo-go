package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStopBeforeServe verifies that Stop() is safe to call before Serve()
// and does not panic.
func TestStopBeforeServe(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)
	// Should not panic when server has not been started
	assert.NotPanics(t, func() {
		srv.Stop()
	})
}

// TestStopIdempotent verifies that calling Stop() multiple times is safe.
func TestStopIdempotent(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	// Start Serve in a goroutine; it will block on stopCh.
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- srv.Serve()
	}()

	// Give Serve() time to reach the blocking phase.
	time.Sleep(200 * time.Millisecond)

	// First Stop should succeed.
	assert.NotPanics(t, func() {
		srv.Stop()
	})

	// Wait for Serve to return.
	select {
	case err := <-serveErrCh:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not return after Stop()")
	}

	// Second Stop should be safe (no-op).
	assert.NotPanics(t, func() {
		srv.Stop()
	})
}

// TestServeStopGracefulShutdown verifies that Stop() causes Serve() to return gracefully.
func TestServeStopGracefulShutdown(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- srv.Serve()
	}()

	// Allow Serve() to proceed past initialization and reach the blocking phase.
	time.Sleep(200 * time.Millisecond)

	// Trigger shutdown.
	srv.Stop()

	// Serve should return without error.
	select {
	case err := <-serveErrCh:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not return after Stop()")
	}
}

// TestServeAfterStop verifies that Serve() can be called again after Stop().
func TestServeAfterStop(t *testing.T) {
	srv, err := NewServer()
	require.NoError(t, err)

	// First cycle
	serveErrCh1 := make(chan error, 1)
	go func() {
		serveErrCh1 <- srv.Serve()
	}()
	time.Sleep(200 * time.Millisecond)
	srv.Stop()
	select {
	case err := <-serveErrCh1:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("first Serve() did not return after Stop()")
	}

	// Second cycle
	serveErrCh2 := make(chan error, 1)
	go func() {
		serveErrCh2 <- srv.Serve()
	}()
	time.Sleep(200 * time.Millisecond)
	srv.Stop()
	select {
	case err := <-serveErrCh2:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("second Serve() did not return after Stop()")
	}
}
