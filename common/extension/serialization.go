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

package extension

import (
	"fmt"
	"io"
	"maps"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

var serializationRegistrationMu sync.Mutex
var serializationIDs = make(map[byte]string)
var serializations = NewRegistry[serialization.Registration]("serialization")
var serializationName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

func serializationKey(protocol, name string) string {
	if protocol == "tri" {
		protocol = "triple"
	}
	if name == "protobuf" {
		name = "proto"
	}
	return protocol + "/" + name
}

func RegisterSerialization(reg serialization.Registration) error {
	serializationRegistrationMu.Lock()
	defer serializationRegistrationMu.Unlock()
	if reg.Protocol != "triple" && reg.Protocol != "dubbo" && reg.Protocol != "grpc" {
		return fmt.Errorf("serialization: invalid protocol %q", reg.Protocol)
	}
	if !serializationName.MatchString(reg.Name) || reg.Name == "protobuf" {
		return fmt.Errorf("serialization: invalid canonical name %q", reg.Name)
	}
	if reg.New == nil {
		return fmt.Errorf("serialization %s: nil factory", reg.Name)
	}
	if reg.Mode < serialization.MessageMode || reg.Mode > serialization.StreamMode {
		return fmt.Errorf("serialization %s: invalid mode", reg.Name)
	}
	if (reg.Protocol == "dubbo") != (reg.Mode == serialization.StreamMode) {
		return fmt.Errorf("serialization %s: mode incompatible with protocol", reg.Name)
	}
	if reg.Protocol != "dubbo" && !serializationName.MatchString(reg.WireName) {
		return fmt.Errorf("serialization %s: invalid wire name", reg.Name)
	}
	if reg.Mode == serialization.MessageMode && reg.WireName != reg.Name {
		return fmt.Errorf("serialization %s: message wire name must equal codec name", reg.Name)
	}
	if reg.Mode == serialization.WrappedObjectMode && (reg.Protocol != "triple" || reg.WireName != "proto") {
		return fmt.Errorf("serialization %s: wrappers require triple/proto", reg.Name)
	}
	if strings.TrimSpace(reg.ConfigPrefix) != reg.ConfigPrefix {
		return fmt.Errorf("serialization %s: invalid config prefix", reg.Name)
	}
	if reg.Protocol == "dubbo" {
		if reg.DubboID == nil || *reg.DubboID == 0 || *reg.DubboID > 31 {
			return fmt.Errorf("serialization %s: Dubbo requires a nonzero 5-bit ID", reg.Name)
		}
		id := *reg.DubboID
		if (id == 2 && reg.Name != "hessian2") || (id == 21 && reg.Name != "proto") {
			return fmt.Errorf("serialization %s: reserved Dubbo ID %d", reg.Name, id)
		}
		if name, ok := serializationIDs[id]; ok {
			return fmt.Errorf("serialization %s: ID %d already registered by %s", reg.Name, id, name)
		}
		reg.DubboID = &id
	} else if reg.DubboID != nil {
		return fmt.Errorf("serialization %s: Dubbo ID on another protocol", reg.Name)
	}

	if !serializations.RegisterIfAbsent(serializationKey(reg.Protocol, reg.Name), reg) {
		return fmt.Errorf("serialization %s/%s already registered", reg.Protocol, reg.Name)
	}
	if reg.DubboID != nil {
		serializationIDs[*reg.DubboID] = reg.Name
	}
	return nil
}
func MustRegisterSerialization(reg serialization.Registration) {
	if err := RegisterSerialization(reg); err != nil {
		panic(err)
	}
}
func LookupSerialization(protocol, name string) (serialization.Registration, bool) {
	reg, ok := serializations.Get(serializationKey(protocol, name))
	if reg.DubboID != nil {
		id := *reg.DubboID
		reg.DubboID = &id
	}
	return reg, ok
}

// BuildSerialization validates a registration and constructs one runtime binding.
// It never publishes the configured instance into the process registry.
func BuildSerialization(reg serialization.Registration, opts serialization.Options) (result serialization.Codec, err error) {
	if reg.Mode < serialization.MessageMode || reg.Mode > serialization.StreamMode {
		return nil, fmt.Errorf("serialization %s: invalid mode", reg.Name)
	}
	if reg.New == nil {
		return nil, fmt.Errorf("serialization %s: nil factory", reg.Name)
	}
	opts.Parameters = maps.Clone(opts.Parameters)
	opts.Protocol = reg.Protocol
	c, err := reg.New(opts)
	defer func() {
		if err != nil && !serializationCodecIsNil(c) {
			if closer, ok := c.(io.Closer); ok {
				_ = closer.Close()
			}
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("serialization %s/%s: %w", reg.Protocol, reg.Name, err)
	}
	if serializationCodecIsNil(c) {
		return nil, fmt.Errorf("serialization %s: factory returned nil", reg.Name)
	}
	if c.Name() != reg.Name {
		return nil, fmt.Errorf("serialization %s: factory returned name %q", reg.Name, c.Name())
	}
	if reg.Mode == serialization.StreamMode {
		if _, ok := c.(serialization.StreamCodec); !ok {
			return nil, fmt.Errorf("serialization %s: missing StreamCodec capability", reg.Name)
		}
	}
	return c, nil
}

// SerializationConfigAttribute is local runtime state; never publish it in URLs.
const SerializationConfigAttribute = "serialization.configs"

// InitializationResult contains scope-local, independently owned snapshots.
type InitializationResult struct {
	FilterNames  []string
	CodecConfigs map[string]any
}

func serializationCodecIsNil(c serialization.Codec) bool {
	if c == nil {
		return true
	}
	v := reflect.ValueOf(c)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}
