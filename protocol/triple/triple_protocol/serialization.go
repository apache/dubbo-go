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
	"maps"
)

// BuiltinSerializationCodecs returns fresh bindings for the built-in formats.
func BuiltinSerializationCodecs() map[string]Codec {
	return map[string]Codec{"proto": &protoBinaryCodec{}, "json": &protoJSONCodec{name: "json"}, "hessian2": &hessian2Codec{}, "msgpack": &msgpackCodec{}}
}

// ConfiguredSerializationOptions binds validated codecs to a client or handler.
// Maps are copied; callers must treat codec instances as immutable bindings.
func ConfiguredSerializationOptions(selected Codec, inner, messages map[string]Codec, wrapped, acceptResponses bool) ([]ClientOption, []HandlerOption) {
	clone := func(source map[string]Codec) map[string]Codec {
		out := make(map[string]Codec, len(source))
		maps.Copy(out, source)
		return out
	}
	inner, messages = clone(inner), clone(messages)
	if wrapped {
		allowed := map[string]Codec{selected.Name(): selected}
		if acceptResponses {
			allowed = inner
		}
		selected = &protoWrapperCodec{innerCodec: selected, runtimeCodecs: allowed}
	}
	return []ClientOption{WithCodec(selected)}, []HandlerOption{&serializationHandlerOption{inner: inner, messages: messages}}
}

type serializationHandlerOption struct{ inner, messages map[string]Codec }

func (o *serializationHandlerOption) applyToHandler(c *handlerConfig) {
	c.InnerCodecs = o.inner
	c.MessageCodecs = o.messages
	maps.Copy(c.Codecs, o.messages)
}

func canonicalInnerName(name string) string {
	if name == "" || name == "hessian4" {
		return "hessian2"
	}
	return name
}
func resolveRuntimeInner(codecs map[string]Codec, name string) (Codec, error) {
	name = canonicalInnerName(name)
	if codecs == nil {
		return resolveInnerCodec(name)
	}
	c, ok := codecs[name]
	if !ok {
		return nil, fmt.Errorf("unsupported or disabled serialize type %q", name)
	}
	return c, nil
}

// SerializationArguments carries invocation-declared parameter names through
// the client wrapper without attaching type metadata to shared codec state.
type SerializationArguments struct {
	Values    []any
	TypeNames []string
}
