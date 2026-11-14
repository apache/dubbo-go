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

package triple_protocol

import (
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

// WithSerializationResources ties codec ownership to complete handler calls and
// client streams, including decode and response encode outside interceptors.
func WithSerializationResources(resources *serialization.Resources) Option {
	return &serializationResourcesOption{resources}
}

type serializationResourcesOption struct{ resources *serialization.Resources }

func (o *serializationResourcesOption) applyToClient(c *clientConfig) { c.CodecResources = o.resources }
func (o *serializationResourcesOption) applyToHandler(c *handlerConfig) {
	c.CodecResources = o.resources
}

type serializationClientConn struct {
	StreamingClientConn
	resources  *serialization.Resources
	once       sync.Once
	operations sync.RWMutex
	closed     bool
}

func (c *serializationClientConn) release() {
	c.once.Do(func() { c.operations.Lock(); defer c.operations.Unlock(); c.closed = true; c.resources.End() })
}
func (c *serializationClientConn) Send(v any) error {
	c.operations.RLock()
	defer c.operations.RUnlock()
	if c.closed {
		return errorf(CodeUnavailable, "serialization stream closed")
	}
	return c.StreamingClientConn.Send(v)
}
func (c *serializationClientConn) Receive(v any) error {
	c.operations.RLock()
	if c.closed {
		c.operations.RUnlock()
		return errorf(CodeUnavailable, "serialization stream closed")
	}
	err := c.StreamingClientConn.Receive(v)
	c.operations.RUnlock()
	if err != nil {
		c.release()
	}
	return err
}
func (c *serializationClientConn) CloseResponse() error {
	err := c.StreamingClientConn.CloseResponse()
	c.release()
	return err
}
