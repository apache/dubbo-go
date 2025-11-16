/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package connection

import (
	"context"
	"testing"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/stretchr/testify/assert"
)

// MockConnection implements Connection interface for testing
type MockConnection struct {
	url        *common.URL
	state      ConnectionState
	available  bool
	lastActive time.Time
	protocol   string
}

func NewMockConnection(url *common.URL, protocol string) *MockConnection {
	return &MockConnection{
		url:        url,
		state:      StateReady,
		available:  true,
		lastActive: time.Now(),
		protocol:   protocol,
	}
}

func (mc *MockConnection) GetState() ConnectionState {
	return mc.state
}

func (mc *MockConnection) GetURL() *common.URL {
	return mc.url
}

func (mc *MockConnection) IsAvailable() bool {
	return mc.available
}

func (mc *MockConnection) Close() error {
	mc.state = StateShutdown
	mc.available = false
	return nil
}

func (mc *MockConnection) GetLastActive() time.Time {
	return mc.lastActive
}

func (mc *MockConnection) GetProtocol() string {
	return mc.protocol
}

func (mc *MockConnection) SetState(state ConnectionState) {
	mc.state = state
}

func (mc *MockConnection) SetAvailable(available bool) {
	mc.available = available
}

func (mc *MockConnection) SetLastActive(t time.Time) {
	mc.lastActive = t
}

// MockConnectionPool implements ConnectionPool interface for testing
type MockConnectionPool struct {
	connections map[string]*MockConnection
	stats       *ConnectionStats
	protocol    string
}

func NewMockConnectionPool(protocol string) *MockConnectionPool {
	return &MockConnectionPool{
		connections: make(map[string]*MockConnection),
		stats: &ConnectionStats{
			TotalConnections:  0,
			ActiveConnections: 0,
		},
		protocol: protocol,
	}
}

func (mcp *MockConnectionPool) GetConnection(url *common.URL) (Connection, error) {
	key := url.Location
	if conn, exists := mcp.connections[key]; exists {
		if conn.IsAvailable() {
			return conn, nil
		}
		// Remove stale connection
		delete(mcp.connections, key)
		mcp.stats.TotalConnections--
	}

	// Create new connection
	conn := NewMockConnection(url, mcp.protocol)
	mcp.connections[key] = conn
	mcp.stats.TotalConnections++
	mcp.stats.ActiveConnections++

	return conn, nil
}

func (mcp *MockConnectionPool) RemoveConnection(conn Connection) error {
	key := conn.GetURL().Location
	if _, exists := mcp.connections[key]; exists {
		delete(mcp.connections, key)
		mcp.stats.TotalConnections--
		mcp.stats.ActiveConnections--
	}
	return nil
}

func (mcp *MockConnectionPool) RemoveStaleConnections(url *common.URL) int {
	removed := 0
	for key, conn := range mcp.connections {
		if !conn.IsAvailable() || time.Since(conn.GetLastActive()) > 5*time.Minute {
			delete(mcp.connections, key)
			removed++
			mcp.stats.TotalConnections--
		}
	}
	return removed
}

func (mcp *MockConnectionPool) GetStats() *ConnectionStats {
	return mcp.stats
}

func (mcp *MockConnectionPool) Close() error {
	for _, conn := range mcp.connections {
		conn.Close()
	}
	mcp.connections = make(map[string]*MockConnection)
	mcp.stats.TotalConnections = 0
	mcp.stats.ActiveConnections = 0
	return nil
}

// MockHealthChecker implements HealthChecker interface for testing
type MockHealthChecker struct {
	protocol      string
	failHealthy   bool
	checkInterval time.Duration
}

func NewMockHealthChecker(protocol string) *MockHealthChecker {
	return &MockHealthChecker{
		protocol:      protocol,
		failHealthy:   false,
		checkInterval: 10 * time.Second,
	}
}

func (mhc *MockHealthChecker) CheckConnection(ctx context.Context, conn Connection) *HealthCheckResult {
	result := &HealthCheckResult{
		Healthy:   true,
		CheckTime: time.Now(),
		Duration:  10 * time.Millisecond,
	}

	if mhc.failHealthy {
		result.Healthy = false
		result.Reason = "mock health check failure"
		return result
	}

	if !conn.IsAvailable() {
		result.Healthy = false
		result.Reason = "connection not available"
		return result
	}

	if conn.GetState() != StateReady {
		result.Healthy = false
		result.Reason = "connection not ready"
		return result
	}

	return result
}

func (mhc *MockHealthChecker) GetProtocol() string {
	return mhc.protocol
}

func (mhc *MockHealthChecker) GetCheckInterval() time.Duration {
	return mhc.checkInterval
}

func (mhc *MockHealthChecker) SetFailHealthy(fail bool) {
	mhc.failHealthy = fail
}

// Test the unified connection manager
func TestUnifiedConnectionManager(t *testing.T) {
	t.Log("Testing Unified Connection Manager - Issue #1868 framework solution")

	// Create manager
	config := DefaultManagerConfig()
	config.HealthCheckInterval = 100 * time.Millisecond // Fast for testing
	manager := NewUnifiedConnectionManager(config)

	// Register protocols
	dubboPool := NewMockConnectionPool("dubbo")
	dubboChecker := NewMockHealthChecker("dubbo")
	err := manager.RegisterProtocol("dubbo", dubboPool, dubboChecker)
	assert.NoError(t, err)

	triplePool := NewMockConnectionPool("triple")
	tripleChecker := NewMockHealthChecker("triple")
	err = manager.RegisterProtocol("triple", triplePool, tripleChecker)
	assert.NoError(t, err)

	// Test getting connections
	dubboURL, _ := common.NewURL("dubbo://127.0.0.1:20880/test")
	tripleURL, _ := common.NewURL("triple://127.0.0.1:50051/test")

	// Get Dubbo connection
	dubboConn, err := manager.GetConnection(dubboURL)
	assert.NoError(t, err)
	assert.NotNil(t, dubboConn)
	assert.Equal(t, "dubbo", dubboConn.GetProtocol())

	// Get Triple connection
	tripleConn, err := manager.GetConnection(tripleURL)
	assert.NoError(t, err)
	assert.NotNil(t, tripleConn)
	assert.Equal(t, "triple", tripleConn.GetProtocol())

	// Test health checking
	assert.True(t, manager.IsConnectionHealthy(dubboConn))
	assert.True(t, manager.IsConnectionHealthy(tripleConn))

	t.Log("Unified connection manager basic functionality works")
}

func TestConnectionHealthManagement(t *testing.T) {
	t.Log("Testing connection health management for Issue #1868")

	manager := NewUnifiedConnectionManager(DefaultManagerConfig())

	// Register Dubbo protocol
	dubboPool := NewMockConnectionPool("dubbo")
	dubboChecker := NewMockHealthChecker("dubbo")
	manager.RegisterProtocol("dubbo", dubboPool, dubboChecker)

	url, _ := common.NewURL("dubbo://127.0.0.1:20880/test")

	// Get healthy connection
	conn, err := manager.GetConnection(url)
	assert.NoError(t, err)
	assert.True(t, manager.IsConnectionHealthy(conn))

	// Simulate connection becoming unhealthy
	mockConn := conn.(*MockConnection)
	mockConn.SetAvailable(false)

	// Health check should fail
	assert.False(t, manager.IsConnectionHealthy(conn))

	// Getting connection should trigger cleanup and create new one
	newConn, err := manager.GetConnection(url)
	assert.NoError(t, err)
	assert.NotEqual(t, conn, newConn) // Should be a different connection
	assert.True(t, manager.IsConnectionHealthy(newConn))

	t.Log("Connection health management works correctly")
}

func TestStaleConnectionCleanup(t *testing.T) {
	t.Log("Testing stale connection cleanup for Issue #1868")

	manager := NewUnifiedConnectionManager(DefaultManagerConfig())

	dubboPool := NewMockConnectionPool("dubbo")
	dubboChecker := NewMockHealthChecker("dubbo")
	manager.RegisterProtocol("dubbo", dubboPool, dubboChecker)

	url, _ := common.NewURL("dubbo://127.0.0.1:20880/test")

	// Create connection and make it stale
	conn, err := manager.GetConnection(url)
	assert.NoError(t, err)

	mockConn := conn.(*MockConnection)
	mockConn.SetLastActive(time.Now().Add(-10 * time.Minute)) // Make it very old

	// Remove stale connections
	removed := manager.RemoveStaleConnections(url)
	assert.Equal(t, 1, removed)

	t.Log("Stale connection cleanup works correctly")
}

func TestMultiProtocolStats(t *testing.T) {
	t.Log("Testing multi-protocol statistics")

	manager := NewUnifiedConnectionManager(DefaultManagerConfig())

	// Register multiple protocols
	dubboPool := NewMockConnectionPool("dubbo")
	dubboChecker := NewMockHealthChecker("dubbo")
	manager.RegisterProtocol("dubbo", dubboPool, dubboChecker)

	triplePool := NewMockConnectionPool("triple")
	tripleChecker := NewMockHealthChecker("triple")
	manager.RegisterProtocol("triple", triplePool, tripleChecker)

	// Create connections
	dubboURL, _ := common.NewURL("dubbo://127.0.0.1:20880/test")
	tripleURL, _ := common.NewURL("triple://127.0.0.1:50051/test")

	manager.GetConnection(dubboURL)
	manager.GetConnection(tripleURL)

	// Check global stats
	stats := manager.GetGlobalStats()
	assert.Contains(t, stats, "dubbo")
	assert.Contains(t, stats, "triple")
	assert.Equal(t, 1, stats["dubbo"].TotalConnections)
	assert.Equal(t, 1, stats["triple"].TotalConnections)

	t.Log("Multi-protocol statistics work correctly")
}

// Test event listening
func TestConnectionEventListening(t *testing.T) {
	t.Log("Testing connection event listening")

	manager := NewUnifiedConnectionManager(DefaultManagerConfig())

	// Create event listener
	events := make([]string, 0)
	listener := &testEventListener{events: &events}
	manager.AddListener(listener)

	dubboPool := NewMockConnectionPool("dubbo")
	dubboChecker := NewMockHealthChecker("dubbo")
	dubboChecker.SetFailHealthy(true) // Force health check failures
	manager.RegisterProtocol("dubbo", dubboPool, dubboChecker)

	url, _ := common.NewURL("dubbo://127.0.0.1:20880/test")

	// Get connection and check health (should fail)
	conn, err := manager.GetConnection(url)
	assert.NoError(t, err)

	healthy := manager.IsConnectionHealthy(conn)
	assert.False(t, healthy) // Should fail due to mock checker

	// Should have received health check failed event
	assert.True(t, len(*listener.events) > 0)

	t.Log("Connection event listening works correctly")
}

type testEventListener struct {
	events *[]string
}

func (tel *testEventListener) OnStateChange(conn Connection, oldState, newState ConnectionState) {
	*tel.events = append(*tel.events, "state_change")
}

func (tel *testEventListener) OnHealthCheckFailed(conn Connection, result *HealthCheckResult) {
	*tel.events = append(*tel.events, "health_check_failed")
}

func (tel *testEventListener) OnConnectionRemoved(conn Connection, reason string) {
	*tel.events = append(*tel.events, "connection_removed")
}
