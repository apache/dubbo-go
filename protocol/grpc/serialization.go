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

package grpc

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

import (
	"github.com/golang/protobuf/jsonpb"

	"google.golang.org/grpc/encoding"

	"google.golang.org/protobuf/proto"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

type protoSerialization struct{}

func (protoSerialization) Name() string { return "proto" }
func (protoSerialization) Marshal(v any) ([]byte, error) {
	p, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("grpc proto: invalid message %T", v)
	}
	return proto.Marshal(p)
}
func (protoSerialization) Unmarshal(b []byte, v any) error {
	p, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("grpc proto: invalid message %T", v)
	}
	return proto.Unmarshal(b, p)
}
func init() {
	for _, c := range []serialization.Codec{protoSerialization{}, grpcJson{Marshaler: jsonpb.Marshaler{EmitDefaults: true, OrigName: true}}} {
		extension.MustRegisterSerialization(serialization.Registration{Protocol: "grpc", Name: c.Name(), Mode: serialization.MessageMode, WireName: c.Name(), New: func(serialization.Options) (serialization.Codec, error) { return c, nil }})
	}
}

type installedSerialization struct {
	config     any
	parameters map[string]string
	codec      serialization.Codec
}

var grpcSerializations = struct {
	sync.Mutex
	items map[string]installedSerialization
}{items: make(map[string]installedSerialization)}

// PrepareCodec uses grpc-go's process-wide subtype registry. Factories receive
// no endpoint/side identity: one subtype must mean the same format process-wide.
func (gp *GrpcProtocol) PrepareCodec(url *common.URL, side string) error {
	name := url.GetParam("serialization", "proto")
	if name == "protobuf" {
		name = "proto"
	}
	reg, ok := extension.LookupSerialization("grpc", name)
	if !ok {
		return fmt.Errorf("unsupported grpc serialization %q", name)
	}
	if name == "proto" || name == "json" {
		return nil
	}
	var config any
	if raw, ok := url.GetAttribute(extension.SerializationConfigAttribute); ok {
		if configs, ok := raw.(map[string]any); ok {
			config = configs[reg.ConfigPrefix]
		}
	}
	parameters := make(map[string]string)
	for key, values := range url.CopyParams() {
		if reg.ConfigPrefix != "" && strings.HasPrefix(key, reg.ConfigPrefix+".") && len(values) > 0 {
			parameters[key] = values[0]
		}
	}
	grpcSerializations.Lock()
	defer grpcSerializations.Unlock()
	if installed, ok := grpcSerializations.items[name]; ok {
		if !reflect.DeepEqual(installed.config, config) || !reflect.DeepEqual(installed.parameters, parameters) {
			return fmt.Errorf("grpc subtype %q already uses different configuration", name)
		}
		return nil
	}
	if encoding.GetCodec(name) != nil || encoding.GetCodecV2(name) != nil {
		return fmt.Errorf("grpc subtype %q is already registered outside the serialization SPI", name)
	}
	codec, err := extension.BuildSerialization(reg, serialization.Options{Protocol: "grpc", Config: config, Parameters: parameters})
	if err != nil {
		return err
	}
	// Codec has exactly grpc-go's v1 adapter contract. Do not force a server
	// codec: health/reflection and other registered subtypes remain available.
	encoding.RegisterCodec(codec)
	grpcSerializations.items[name] = installedSerialization{config: config, parameters: parameters, codec: codec}
	return nil
}
