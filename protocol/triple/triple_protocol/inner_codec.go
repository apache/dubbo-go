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
	"fmt"
)

// innerCodecRegistry is the allowed-set of Triple non-IDL inner serialization
// codecs. The name key MUST match the SerializeType string the client writes
// into TripleRequestWrapper.SerializeType / TripleResponseWrapper.SerializeType
// (e.g. "hessian2", "msgpack"). An absent entry is a disabled serialization.
type innerCodecRegistry struct {
	items map[string]Codec
}

var innerCodecs = &innerCodecRegistry{items: make(map[string]Codec)}

// SetInnerCodec registers an inner codec under the given name.
//
// SetInnerCodec is init-only: it MUST be called exclusively from package init().
func SetInnerCodec(name string, c Codec) {
	innerCodecs.items[name] = c
}

// GetInnerCodec looks up a registered inner codec by name.
// Returns (nil, false) when the name is unknown or disabled (unregistered).
func GetInnerCodec(name string) (Codec, bool) {
	c, ok := innerCodecs.items[name]
	return c, ok
}

// innerCodecNames returns the registered inner codec names, for error
// diagnostics. Order is unspecified.
func innerCodecNames() []string {
	names := make([]string, 0, len(innerCodecs.items))
	for n := range innerCodecs.items {
		names = append(names, n)
	}
	return names
}

// resolveInnerCodec looks up the inner codec registered under serializeType
// in the inner codec registry. An empty serializeType defaults to hessian2 for
// backward compatibility.
func resolveInnerCodec(serializeType string) (Codec, error) {
	if serializeType == "" {
		serializeType = codecNameHessian2
	}
	c, ok := GetInnerCodec(serializeType)
	if !ok {
		return nil, fmt.Errorf("unsupported or disabled serialize type %q (registered: %v)",
			serializeType, innerCodecNames())
	}
	return c, nil
}

func init() {
	SetInnerCodec(codecNameHessian2, &hessian2Codec{})
	SetInnerCodec(codecNameMsgPack, &msgpackCodec{})
}
