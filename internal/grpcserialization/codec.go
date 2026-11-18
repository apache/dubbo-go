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

// Package grpcserialization reserves grpc-go adapters before RPCs start.
package grpcserialization

import (
	"fmt"
	"sync"
)

import (
	"google.golang.org/grpc/encoding"
)

import (
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

var registry = struct {
	sync.Mutex
	frozen   bool
	adapters map[string]*Adapter
}{adapters: make(map[string]*Adapter)}

// Register is called with the public SPI descriptor during initialization.
// grpc-go's own registry must never be modified after RPC traffic starts.
func Register(name string) error {
	registry.Lock()
	defer registry.Unlock()
	if registry.frozen {
		return fmt.Errorf("grpc serialization %q: register descriptors during init, before starting RPCs", name)
	}
	if encoding.GetCodec(name) != nil || encoding.GetCodecV2(name) != nil {
		// Preserve the external registration; PrepareCodec reports the conflict.
		return nil
	}
	adapter := &Adapter{name: name}
	encoding.RegisterCodec(adapter)
	registry.adapters[name] = adapter
	return nil
}

// Freeze ends descriptor registration before clients or servers begin RPCs.
func Freeze() {
	registry.Lock()
	registry.frozen = true
	registry.Unlock()
}

func Lookup(name string) *Adapter {
	registry.Lock()
	defer registry.Unlock()
	adapter := registry.adapters[name]
	if adapter == nil || encoding.GetCodec(name) != adapter || encoding.GetCodecV2(name) != nil {
		return nil
	}
	return adapter
}

// Adapter remains in grpc-go's registry for the process lifetime. Only its
// configured delegate changes, under a lock independent of grpc-go's readers.
type Adapter struct {
	name  string
	mu    sync.RWMutex
	codec serialization.Codec
}

func (a *Adapter) Name() string { return a.name }

func (a *Adapter) Bind(codec serialization.Codec) {
	a.mu.Lock()
	a.codec = codec
	a.mu.Unlock()
}

func (a *Adapter) delegate() (serialization.Codec, error) {
	a.mu.RLock()
	codec := a.codec
	a.mu.RUnlock()
	if codec == nil {
		return nil, fmt.Errorf("grpc subtype %q is not configured", a.name)
	}
	return codec, nil
}

func (a *Adapter) Marshal(message any) ([]byte, error) {
	codec, err := a.delegate()
	if err != nil {
		return nil, err
	}
	return codec.Marshal(message)
}

func (a *Adapter) Unmarshal(data []byte, message any) error {
	codec, err := a.delegate()
	if err != nil {
		return err
	}
	return codec.Unmarshal(data, message)
}
