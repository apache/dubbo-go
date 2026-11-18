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
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo/impl"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/serialization"
	hessian "dubbo.apache.org/dubbo-go/v3/serialization/hessian2"
)

func init() {
	id := byte(2)
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "dubbo", Name: "hessian2", Mode: serialization.StreamMode, DubboID: &id, New: func(serialization.Options) (serialization.Codec, error) { return &hessian.Codec{}, nil }})
}
func newConfiguredDubboCodec(url *common.URL) (result remoting.Codec, err error) {
	resources := &serialization.Resources{}
	defer func() {
		if err != nil {
			_ = resources.Close()
		}
	}()
	name := url.GetParam("serialization", "hessian2")
	names := []string{"hessian2", name}
	if url.GetParam("side", "") == "consumer" {
		names = []string{name}
	}
	if namesOverride, explicit := extension.AcceptedSerializations(url); explicit {
		names = namesOverride
	}

	accepted := make(map[string]bool)
	for _, n := range names {
		accepted[n] = true
	}
	names = append(names, name)
	c := &DubboCodec{resources: resources, serializers: make(map[byte]impl.Serializer), names: make(map[string]byte)}
	for _, name := range names {
		if name == "" {
			continue
		}
		if _, ok := c.names[name]; ok {
			continue
		}
		reg, ok := extension.LookupSerialization("dubbo", name)
		if !ok {
			return nil, fmt.Errorf("unsupported Dubbo serialization %q", name)
		}
		params := make(map[string]string)
		for key, values := range url.CopyParams() {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}
		var config any
		if raw, ok := url.GetAttribute(extension.SerializationConfigAttribute); ok {
			if snapshots, ok := raw.(map[string]any); ok {
				config = snapshots[reg.ConfigPrefix]
			}
		}
		codec, err := extension.BuildSerialization(reg, serialization.Options{Protocol: "dubbo", Side: url.GetParam("side", ""), Parameters: params, Config: config})
		if err != nil {
			return nil, err
		}
		resources.Add(codec)
		var serializer impl.Serializer = impl.StreamSerializer{Codec: codec.(serialization.StreamCodec)}
		// Preserve applications which use the legacy complete-body Serializer SPI.
		if name == "hessian2" {
			legacy, _ := impl.GetSerializerById(2)
			switch legacy.(type) {
			case impl.HessianSerializer, *impl.HessianSerializer:
			default:
				serializer = legacy
			}
		}
		if accepted[name] {
			c.serializers[*reg.DubboID] = serializer
		}
		if name == url.GetParam("serialization", "hessian2") {
			c.sendSerializer = serializer
		}
		c.names[name] = *reg.DubboID
	}
	c.selectedID = c.names[name]
	return c, nil
}
