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
	"time"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// DubboConnection adapts Dubbo's ExchangeClient to the unified Connection interface
// This provides a unified interface for Issue #1868 health checking
type DubboConnection struct {
	client     *remoting.ExchangeClient
	url        *common.URL
	lastActive time.Time
}

// NewDubboConnection creates a new DubboConnection wrapper
func NewDubboConnection(client *remoting.ExchangeClient, url *common.URL) *DubboConnection {
	return &DubboConnection{
		client:     client,
		url:        url,
		lastActive: time.Now(),
	}
}

// GetState returns the current state of the connection
func (dc *DubboConnection) GetState() ConnectionState {
	if dc.client == nil {
		return StateShutdown
	}

	// Map ExchangeClient availability to our state model
	if dc.client.IsAvailable() {
		return StateReady
	}

	// If not available, we need to determine if it's a transient failure
	// or if the connection is being established
	return StateTransientFailure
}

// GetURL returns the URL associated with this connection
func (dc *DubboConnection) GetURL() *common.URL {
	return dc.url
}

// IsAvailable checks if the connection is available for use
func (dc *DubboConnection) IsAvailable() bool {
	if dc.client == nil {
		return false
	}

	// Delegate to the ExchangeClient's IsAvailable method
	// This uses our improved health checking from the ExchangeClient implementation
	return dc.client.IsAvailable()
}

// Close closes the connection
func (dc *DubboConnection) Close() error {
	if dc.client != nil {
		dc.client.Close()
	}
	return nil
}

// GetLastActive returns the last active time of the connection
func (dc *DubboConnection) GetLastActive() time.Time {
	return dc.lastActive
}

// UpdateLastActive updates the last active time
func (dc *DubboConnection) UpdateLastActive() {
	dc.lastActive = time.Now()
}

// GetProtocol returns the protocol name
func (dc *DubboConnection) GetProtocol() string {
	return "dubbo"
}

// GetExchangeClient returns the underlying ExchangeClient
// This allows protocol-specific operations while maintaining the unified interface
func (dc *DubboConnection) GetExchangeClient() *remoting.ExchangeClient {
	return dc.client
}

