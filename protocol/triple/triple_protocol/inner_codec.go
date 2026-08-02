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
	"sort"
)

// innerCodecRegistry is the allowed-set of Triple non-IDL inner serialization
// codecs. The name key MUST match the SerializeType string the client writes
// into TripleRequestWrapper.SerializeType / TripleResponseWrapper.SerializeType
// (e.g. "hessian2", "msgpack"). An absent entry is a disabled serialization.
type innerCodecRegistry struct {
	items map[string]Codec
}

var innerCodecs = &innerCodecRegistry{items: make(map[string]Codec)}

func init() {
	registerInnerCodec(codecNameHessian2, &hessian2Codec{})
	registerInnerCodec(codecNameMsgPack, &msgpackCodec{})
}

// registerInnerCodec registers an inner codec under the given name.
func registerInnerCodec(name string, c Codec) {
	if c == nil {
		panic(fmt.Sprintf("triple_protocol: registerInnerCodec(%q): nil codec", name))
	}
	if c.Name() != name {
		panic(fmt.Sprintf("triple_protocol: registerInnerCodec(%q): codec Name() = %q, must match the registered name",
			name, c.Name()))
	}
	innerCodecs.items[name] = c
}

// getInnerCodec looks up a registered inner codec by name.
// Returns (nil, false) when the name is unknown or disabled (unregistered).
func getInnerCodec(name string) (Codec, bool) {
	c, ok := innerCodecs.items[name]
	return c, ok
}

// innerCodecNames returns the registered inner codec names, sorted for stable
// error diagnostics (map iteration order is unspecified).
func innerCodecNames() []string {
	names := make([]string, 0, len(innerCodecs.items))
	for n := range innerCodecs.items {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// resolveInnerCodec looks up the inner codec registered under serializeType
// in the inner codec registry. An empty serializeType defaults to hessian2 for
// backward compatibility.
//
// Dubbo Java writes "hessian4" into the wrapper (TripleConstants.HESSIAN4)
// while its on-wire encoding is Hessian2-compatible; the Java receiver maps it
// back to "hessian2" in ReflectionPackableMethod.convertHessianFromWrapper. Go
// mirrors that single alias so a Java non-IDL client is not rejected.
func resolveInnerCodec(serializeType string) (Codec, error) {
	if serializeType == "" || serializeType == "hessian4" {
		serializeType = codecNameHessian2
	}
	c, ok := getInnerCodec(serializeType)
	if !ok {
		return nil, fmt.Errorf("unsupported or disabled serialize type %q (registered: %v)",
			serializeType, innerCodecNames())
	}
	return c, nil
}
