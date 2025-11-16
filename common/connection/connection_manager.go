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
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// ConnectionState represents the state of a connection
type ConnectionState int32

const (
	// StateIdle indicates the connection is idle
	StateIdle ConnectionState = iota
	// StateConnecting indicates the connection is being established
	StateConnecting
	// StateReady indicates the connection is ready for use
	StateReady
	// StateTransientFailure indicates the connection has failed temporarily
	StateTransientFailure
	// StateShutdown indicates the connection has been shutdown
	StateShutdown
)

// String returns the string representation of ConnectionState
func (cs ConnectionState) String() string {
	switch cs {
	case StateIdle:
		return "IDLE"
	case StateConnecting:
		return "CONNECTING"
	case StateReady:
		return "READY"
	case StateTransientFailure:
		return "TRANSIENT_FAILURE"
	case StateShutdown:
		return "SHUTDOWN"
	default:
		return "UNKNOWN"
	}
}

// Connection represents a unified connection interface across all protocols
type Connection interface {
	// GetState returns the current state of the connection
	GetState() ConnectionState

	// GetURL returns the URL associated with this connection
	GetURL() *common.URL

	// IsAvailable checks if the connection is available for use
	IsAvailable() bool

	// Close closes the connection
	Close() error

	// GetLastActive returns the last active time of the connection
	GetLastActive() time.Time

	// GetProtocol returns the protocol name (dubbo, triple, grpc, etc.)
	GetProtocol() string
}

// ConnectionStats provides statistics about connections
type ConnectionStats struct {
	TotalConnections  int
	ActiveConnections int
	IdleConnections   int
	FailedConnections int
	ReconnectAttempts int
	LastReconnectTime time.Time
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Healthy   bool
	Reason    string
	CheckTime time.Time
	Duration  time.Duration
}

// HealthChecker defines the interface for connection health checking
type HealthChecker interface {
	// CheckConnection performs health check on a specific connection
	CheckConnection(ctx context.Context, conn Connection) *HealthCheckResult

	// GetProtocol returns the protocol this health checker supports
	GetProtocol() string

	// GetCheckInterval returns the recommended check interval
	GetCheckInterval() time.Duration
}

// ConnectionPool manages connections for a specific protocol
type ConnectionPool interface {
	// GetConnection retrieves a healthy connection for the given URL
	GetConnection(url *common.URL) (Connection, error)

	// RemoveConnection removes a connection from the pool
	RemoveConnection(conn Connection) error

	// RemoveStaleConnections removes all stale connections for the given URL
	RemoveStaleConnections(url *common.URL) int

	// GetStats returns connection pool statistics
	GetStats() *ConnectionStats

	// Close closes all connections in the pool
	Close() error
}

// ConnectionManager provides unified connection management across all protocols
type ConnectionManager interface {
	// RegisterProtocol registers a connection pool and health checker for a protocol
	RegisterProtocol(protocol string, pool ConnectionPool, checker HealthChecker) error

	// GetConnection gets a healthy connection for the specified URL
	GetConnection(url *common.URL) (Connection, error)

	// IsConnectionHealthy checks if the connection is healthy
	IsConnectionHealthy(conn Connection) bool

	// RemoveStaleConnections removes stale connections for the given URL
	RemoveStaleConnections(url *common.URL) int

	// GetGlobalStats returns global connection statistics across all protocols
	GetGlobalStats() map[string]*ConnectionStats

	// StartHealthCheckLoop starts the background health check loop
	StartHealthCheckLoop(ctx context.Context)

	// Close closes all connections across all protocols
	Close() error
}

// StateChangeCallback is called when connection state changes
type StateChangeCallback func(conn Connection, oldState, newState ConnectionState)

// ConnectionEventListener listens to connection events
type ConnectionEventListener interface {
	// OnStateChange is called when connection state changes
	OnStateChange(conn Connection, oldState, newState ConnectionState)

	// OnHealthCheckFailed is called when health check fails
	OnHealthCheckFailed(conn Connection, result *HealthCheckResult)

	// OnConnectionRemoved is called when a connection is removed
	OnConnectionRemoved(conn Connection, reason string)
}
