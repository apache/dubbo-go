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

package dubbo

import (
	"fmt"
	"sync"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/connection"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/getty"
	"github.com/dubbogo/gost/log/logger"
)

// DubboConnectionPool implements ConnectionPool for Dubbo protocol
// This integrates the unified connection management framework with actual Dubbo protocol
type DubboConnectionPool struct {
	// Connection storage - migrated from global exchangeClientMap
	connections map[string]*connection.DubboConnection
	mutex       sync.RWMutex

	// Lock management for connection creation
	connectionLocks map[string]*sync.Mutex
	lockMutex       sync.RWMutex

	// Statistics
	stats *connection.ConnectionStats
}

// NewDubboConnectionPool creates a new Dubbo connection pool
func NewDubboConnectionPool() *DubboConnectionPool {
	return &DubboConnectionPool{
		connections:     make(map[string]*connection.DubboConnection),
		connectionLocks: make(map[string]*sync.Mutex),
		stats: &connection.ConnectionStats{
			TotalConnections:  0,
			ActiveConnections: 0,
			IdleConnections:   0,
			FailedConnections: 0,
		},
	}
}

// GetConnection retrieves or creates a healthy connection for the given URL
// This replaces the original getExchangeClient() function with unified framework integration
func (dcp *DubboConnectionPool) GetConnection(url *common.URL) (connection.Connection, error) {
	location := url.Location

	// Try to get existing connection
	dcp.mutex.RLock()
	if conn, exists := dcp.connections[location]; exists {
		dcp.mutex.RUnlock()

		// Check if connection is still healthy
		if conn.IsAvailable() {
			conn.GetExchangeClient().IncreaseActiveNumber()
			logger.Debugf("Reusing healthy connection for %s", location)
			return conn, nil
		}

		// Connection is stale, remove it
		logger.Warnf("Found stale connection for %s, removing from pool", location)
		dcp.removeConnection(location)
	} else {
		dcp.mutex.RUnlock()
	}

	// Create new connection with proper locking
	return dcp.createNewConnection(url)
}

// createNewConnection creates a new connection with proper synchronization
func (dcp *DubboConnectionPool) createNewConnection(url *common.URL) (connection.Connection, error) {
	location := url.Location

	// Get or create lock for this location
	dcp.lockMutex.Lock()
	connLock, exists := dcp.connectionLocks[location]
	if !exists {
		connLock = &sync.Mutex{}
		dcp.connectionLocks[location] = connLock
	}
	dcp.lockMutex.Unlock()

	// Lock for this specific connection
	connLock.Lock()
	defer connLock.Unlock()

	// Double-check if connection was created while waiting for lock
	dcp.mutex.RLock()
	if conn, exists := dcp.connections[location]; exists && conn.IsAvailable() {
		dcp.mutex.RUnlock()
		conn.GetExchangeClient().IncreaseActiveNumber()
		return conn, nil
	}
	dcp.mutex.RUnlock()

	// Create new ExchangeClient with proper timeout configuration
	exchangeClient, err := dcp.createExchangeClient(url)
	if err != nil {
		dcp.updateStats(0, 0, 0, 1) // Failed connection
		return nil, err
	}

	// Wrap ExchangeClient in unified Connection interface
	dubboConn := connection.NewDubboConnection(exchangeClient, url)

	// Store in pool
	dcp.mutex.Lock()
	dcp.connections[location] = dubboConn
	dcp.updateStatsLocked(1, 1, 0, 0) // New total and active connection
	dcp.mutex.Unlock()

	logger.Infof("Created new Dubbo connection for %s", location)
	return dubboConn, nil
}

// createExchangeClient creates the actual ExchangeClient with proper timeout configuration
// This encapsulates the original createNewExchangeClient logic
func (dcp *DubboConnectionPool) createExchangeClient(url *common.URL) (*remoting.ExchangeClient, error) {
	// Get timeout configuration with proper priority: URL > consumer config > default
	var requestTimeout time.Duration = 3 * time.Second
	var connectTimeout time.Duration = 3 * time.Second

	// Try to get timeout from consumer config first for backwards compatibility
	rt := config.GetConsumerConfig().RequestTimeout
	if consumerConfRaw, ok := url.GetAttribute("consumer.config"); ok {
		if consumerConf, ok := consumerConfRaw.(*global.ConsumerConfig); ok {
			rt = consumerConf.RequestTimeout
		}
	}

	if rt != "" {
		if timeout, err := time.ParseDuration(rt); err == nil {
			requestTimeout = timeout
			connectTimeout = timeout
		}
	}

	// Override with URL specific timeout if provided (URL parameter takes precedence)
	if timeoutStr := url.GetParam("timeout", ""); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			requestTimeout = timeout
			connectTimeout = timeout
		}
	}

	// Create Getty client with proper timeout configuration
	client := getty.NewClient(getty.Options{
		ConnectTimeout: connectTimeout,
		RequestTimeout: requestTimeout,
	})

	exchangeClient := remoting.NewExchangeClient(url, client, requestTimeout, false)
	if exchangeClient == nil {
		return nil, fmt.Errorf("failed to create exchange client for %s", url.Location)
	}

	return exchangeClient, nil
}

// RemoveConnection removes a specific connection from the pool
func (dcp *DubboConnectionPool) RemoveConnection(conn connection.Connection) error {
	if dubboConn, ok := conn.(*connection.DubboConnection); ok {
		location := dubboConn.GetURL().Location
		return dcp.removeConnection(location)
	}
	return fmt.Errorf("invalid connection type for Dubbo pool")
}

// removeConnection removes connection by location
func (dcp *DubboConnectionPool) removeConnection(location string) error {
	dcp.mutex.Lock()
	defer dcp.mutex.Unlock()

	if conn, exists := dcp.connections[location]; exists {
		conn.Close()
		delete(dcp.connections, location)
		dcp.updateStatsLocked(-1, -1, 0, 0) // Remove total and active
		logger.Debugf("Removed connection for %s", location)
	}

	return nil
}

// RemoveStaleConnections removes all stale connections for the given URL or all stale connections
func (dcp *DubboConnectionPool) RemoveStaleConnections(url *common.URL) int {
	dcp.mutex.Lock()
	defer dcp.mutex.Unlock()

	removed := 0
	now := time.Now()
	maxIdleTime := 5 * time.Minute // Configurable in the future

	for location, conn := range dcp.connections {
		shouldRemove := false

		// Check if URL matches (if specified)
		if url != nil && conn.GetURL().Location != url.Location {
			continue
		}

		// Check if connection is stale
		if !conn.IsAvailable() {
			shouldRemove = true
			logger.Debugf("Removing unavailable connection: %s", location)
		} else if now.Sub(conn.GetLastActive()) > maxIdleTime {
			shouldRemove = true
			logger.Debugf("Removing idle connection: %s (idle for %v)", location, now.Sub(conn.GetLastActive()))
		}

		if shouldRemove {
			conn.Close()
			delete(dcp.connections, location)
			removed++
			dcp.updateStatsLocked(-1, -1, 0, 0)
		}
	}

	if removed > 0 {
		logger.Infof("Removed %d stale connections from Dubbo pool", removed)
	}

	return removed
}

// GetStats returns current connection pool statistics
func (dcp *DubboConnectionPool) GetStats() *connection.ConnectionStats {
	dcp.mutex.RLock()
	defer dcp.mutex.RUnlock()

	// Update real-time stats
	activeCount := 0
	idleCount := 0

	for _, conn := range dcp.connections {
		if conn.IsAvailable() {
			// Simple heuristic: if last active within 1 minute, consider active
			if time.Since(conn.GetLastActive()) < time.Minute {
				activeCount++
			} else {
				idleCount++
			}
		}
	}

	stats := &connection.ConnectionStats{
		TotalConnections:  len(dcp.connections),
		ActiveConnections: activeCount,
		IdleConnections:   idleCount,
		FailedConnections: dcp.stats.FailedConnections,
		ReconnectAttempts: dcp.stats.ReconnectAttempts,
		LastReconnectTime: dcp.stats.LastReconnectTime,
	}

	return stats
}

// Close closes all connections in the pool
func (dcp *DubboConnectionPool) Close() error {
	dcp.mutex.Lock()
	defer dcp.mutex.Unlock()

	for location, conn := range dcp.connections {
		conn.Close()
		delete(dcp.connections, location)
	}

	// Clear locks
	dcp.lockMutex.Lock()
	dcp.connectionLocks = make(map[string]*sync.Mutex)
	dcp.lockMutex.Unlock()

	// Reset stats
	dcp.stats = &connection.ConnectionStats{}

	logger.Info("Dubbo connection pool closed")
	return nil
}

// updateStats updates statistics safely
func (dcp *DubboConnectionPool) updateStats(totalDelta, activeDelta, idleDelta, failedDelta int) {
	dcp.mutex.Lock()
	defer dcp.mutex.Unlock()
	dcp.updateStatsLocked(totalDelta, activeDelta, idleDelta, failedDelta)
}

// updateStatsLocked updates statistics (caller must hold mutex)
func (dcp *DubboConnectionPool) updateStatsLocked(totalDelta, activeDelta, idleDelta, failedDelta int) {
	dcp.stats.TotalConnections += totalDelta
	dcp.stats.ActiveConnections += activeDelta
	dcp.stats.IdleConnections += idleDelta
	dcp.stats.FailedConnections += failedDelta

	if totalDelta > 0 || activeDelta > 0 || idleDelta > 0 || failedDelta > 0 {
		dcp.stats.LastReconnectTime = time.Now()
		dcp.stats.ReconnectAttempts++
	}
}
