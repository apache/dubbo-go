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

package triple

import (
	"fmt"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

func init() {
	for name, c := range tri.BuiltinSerializationCodecs() {
		mode, wire := serialization.MessageMode, name
		if name == "hessian2" || name == "msgpack" {
			mode, wire = serialization.WrappedObjectMode, "proto"
		}
		extension.MustRegisterSerialization(serialization.Registration{Protocol: "triple", Name: name, Mode: mode, WireName: wire, New: func(serialization.Options) (serialization.Codec, error) { return c, nil }})
	}
}
func SerializationOptions(url *common.URL, side string) (client []tri.ClientOption, handler []tri.HandlerOption, isIDL bool, err error) {
	if raw, ok := url.GetAttribute("triple.serialization." + side); ok {
		if binding, ok := raw.(*serializationBinding); ok {
			binding.mu.Lock()
			defer binding.mu.Unlock()
			return binding.client, binding.handler, binding.isIDL, nil
		}
	}

	name := url.GetParam("serialization", "protobuf")
	reg, ok := extension.LookupSerialization("triple", name)
	if !ok {
		return nil, nil, false, fmt.Errorf("unsupported serialization: %s", name)
	}
	resources := &serialization.Resources{}
	defer func() {
		if err != nil {
			_ = resources.Close()
		}
	}()
	makeCodec := func(r serialization.Registration) (tri.Codec, error) {
		parameters := make(map[string]string)
		for key, values := range url.GetParams() {
			if len(values) > 0 {
				parameters[key] = values[0]
			}
		}
		var config any
		if raw, ok := url.GetAttribute(extension.SerializationConfigAttribute); ok {
			if configs, ok := raw.(map[string]any); ok {
				config = configs[r.ConfigPrefix]
			}
		}
		codec, err := extension.BuildSerialization(r, serialization.Options{Protocol: "triple", Side: side, Parameters: parameters, Config: config})
		if err == nil {
			resources.Add(codec)
		}
		return codec, err
	}
	selected, err := makeCodec(reg)
	if err != nil {
		return nil, nil, false, err
	}
	names := []string{"proto", "json", "hessian2", reg.Name}
	accepted, explicit := extension.AcceptedSerializations(url)
	if explicit {
		names = accepted
	}

	inner := make(map[string]tri.Codec)
	messages := make(map[string]tri.Codec)
	for _, name := range names {
		if name == "" {
			continue
		}
		r, ok := extension.LookupSerialization("triple", name)
		if !ok {
			return nil, nil, false, fmt.Errorf("unsupported accepted serialization: %s", name)
		}
		c := selected
		if r.Name != reg.Name {
			c, err = makeCodec(r)
			if err != nil {
				return nil, nil, false, err
			}
		}
		if r.Mode == serialization.WrappedObjectMode {
			inner[r.Name] = c
		} else {
			messages[r.WireName] = c
		}
	}
	if side == "consumer" && explicit && reg.Mode == serialization.MessageMode {
		if _, ok := messages[reg.WireName]; !ok {
			return nil, nil, false, fmt.Errorf("response serialization %q disabled", reg.Name)
		}
	}
	client, handler = tri.ConfiguredSerializationOptions(selected, inner, messages, reg.Mode == serialization.WrappedObjectMode, side == "consumer" && explicit)
	if resources.HasClosers() {
		client = append(client, tri.WithSerializationResources(resources))
		handler = append(handler, tri.WithSerializationResources(resources))
	}
	url.SetAttribute("triple.serialization.resources."+side, resources)
	return client, handler, reg.Mode == serialization.MessageMode, nil
}

type serializationBinding struct {
	mu        sync.Mutex
	users     int
	retired   bool
	resources *serialization.Resources
	source    *common.URL // immutable local construction settings, before discovery merges
	client    []tri.ClientOption
	handler   []tri.HandlerOption
	isIDL     bool
}

func (tp *TripleProtocol) PrepareCodec(url *common.URL, side string) error {
	if raw, ok := url.GetAttribute("triple.serialization." + side); ok {
		if _, ok := raw.(*serializationBinding); ok {
			return nil
		}
	}
	source := url.Clone()
	c, h, idl, err := SerializationOptions(url, side)
	if err != nil {
		return err
	}
	resources, _ := url.GetAttribute("triple.serialization.resources." + side)
	owner, _ := resources.(*serialization.Resources)
	url.SetAttribute("triple.serialization."+side, &serializationBinding{client: c, handler: h, isIDL: idl, resources: owner, source: source})
	return nil
}

// acquireClientSerialization leases the reference-scoped binding to a provider
// invoker. URL clones share this binding, but no provider owns it exclusively.
func acquireClientSerialization(url *common.URL) ([]tri.ClientOption, bool, *serialization.Resources, func() error, error) {
	if _, ok := url.GetAttribute("triple.serialization.consumer"); !ok {
		if err := (&TripleProtocol{}).PrepareCodec(url, "consumer"); err != nil {
			return nil, false, nil, nil, err
		}
	}
	raw, _ := url.GetAttribute("triple.serialization.consumer")
	binding := raw.(*serializationBinding)
	binding.mu.Lock()
	if binding.retired {
		// Discovery can reconnect after all previous providers disappeared. Build a
		// fresh generation from the same local configuration, not a closed instance.
		fresh := binding.source.Clone()
		fresh.DeleteAttribute("triple.serialization.consumer")
		fresh.DeleteAttribute("triple.serialization.resources.consumer")
		client, handler, idl, err := SerializationOptions(fresh, "consumer")
		if err != nil {
			binding.mu.Unlock()
			return nil, false, nil, nil, err
		}
		owner, _ := fresh.GetAttribute("triple.serialization.resources.consumer")
		binding.client, binding.handler, binding.isIDL = client, handler, idl
		binding.resources, _ = owner.(*serialization.Resources)
		binding.retired = false
	}
	binding.users++
	client, idl, resources := binding.client, binding.isIDL, binding.resources
	binding.mu.Unlock()
	url.SetAttribute("triple.serialization.resources.consumer", resources)
	var once sync.Once
	var releaseErr error
	release := func() error {
		once.Do(func() {
			binding.mu.Lock()
			binding.users--
			last := binding.users == 0
			if last {
				binding.retired = true
			}
			binding.mu.Unlock()
			if last {
				releaseErr = resources.Close()
			}
		})
		return releaseErr
	}
	return client, idl, resources, release, nil
}

func (tp *TripleProtocol) ReleaseCodec(url *common.URL, side string) error {
	if side == "consumer" {
		if raw, ok := url.GetAttribute("triple.serialization.consumer"); ok {
			if binding, ok := raw.(*serializationBinding); ok {
				binding.mu.Lock()
				if binding.users > 0 {
					binding.mu.Unlock()
					return nil
				}
				binding.retired = true
				resources := binding.resources
				binding.mu.Unlock()
				return resources.Close()
			}
		}
	}
	if raw, ok := url.GetAttribute("triple.serialization.resources." + side); ok {
		if resources, ok := raw.(*serialization.Resources); ok {
			return resources.Close()
		}
	}
	return nil
}
