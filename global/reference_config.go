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

package global

import (
	"strconv"
)

// ReferenceConfig is the configuration of service consumer
type ReferenceConfig struct {
	InterfaceName    string            `yaml:"interface"  json:"interface,omitempty" property:"interface"`
	Check            *bool             `yaml:"check"  json:"check,omitempty" property:"check"`
	URL              string            `yaml:"url"  json:"url,omitempty" property:"url"`
	Filter           string            `yaml:"filter" json:"filter,omitempty" property:"filter"`
	Protocol         string            `yaml:"protocol"  json:"protocol,omitempty" property:"protocol"`
	RegistryIDs      []string          `yaml:"registry-ids"  json:"registry-ids,omitempty"  property:"registry-ids"`
	Cluster          string            `yaml:"cluster"  json:"cluster,omitempty" property:"cluster"`
	Loadbalance      string            `yaml:"loadbalance"  json:"loadbalance,omitempty" property:"loadbalance"`
	Retries          string            `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	Group            string            `yaml:"group"  json:"group,omitempty" property:"group"`
	Version          string            `yaml:"version"  json:"version,omitempty" property:"version"`
	Serialization    string            `yaml:"serialization" json:"serialization" property:"serialization"`
	ProvidedBy       string            `yaml:"provided_by"  json:"provided_by,omitempty" property:"provided_by"`
	Methods          []*MethodConfig   `yaml:"methods"  json:"methods,omitempty" property:"methods"`
	Async            bool              `yaml:"async"  json:"async,omitempty" property:"async"`
	Params           map[string]string `yaml:"params"  json:"params,omitempty" property:"params"`
	Generic          string            `yaml:"generic"  json:"generic,omitempty" property:"generic"`
	Sticky           bool              `yaml:"sticky"   json:"sticky,omitempty" property:"sticky"`
	RequestTimeout   string            `yaml:"timeout"  json:"timeout,omitempty" property:"timeout"`
	ForceTag         bool              `yaml:"force.tag"  json:"force.tag,omitempty" property:"force.tag"`
	TracingKey       string            `yaml:"tracing-key" json:"tracing-key,omitempty" propertiy:"tracing-key"`
	MeshProviderPort int               `yaml:"mesh-provider-port" json:"mesh-provider-port,omitempty" propertiy:"mesh-provider-port"`
}

func DefaultReferenceConfig() *ReferenceConfig {
	return &ReferenceConfig{
		// use Triple protocol by default
		//Protocol: "tri",
		Methods: make([]*MethodConfig, 0, 8),
		//Params:   make(map[string]string, 8),
	}
}

type ReferenceOption func(*ReferenceConfig)

func WithReference_InterfaceName(name string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.InterfaceName = name
	}
}

func WithReference_Check(check bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Check = &check
	}
}

func WithReference_URL(url string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.URL = url
	}
}

func WithReference_Filter(filter string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Filter = filter
	}
}

func WithReference_Protocol(protocol string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Protocol = protocol
	}
}

func WithReference_RegistryIDs(registryIDs []string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		if len(registryIDs) >= 0 {
			cfg.RegistryIDs = registryIDs
		}
	}
}

func WithReference_Cluster(cluster string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Cluster = cluster
	}
}

func WithReference_LoadBalance(loadBalance string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Loadbalance = loadBalance
	}
}

func WithReference_Retries(retries int) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Retries = strconv.Itoa(retries)
	}
}

func WithReference_Group(group string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Group = group
	}
}

func WithReference_Version(version string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Version = version
	}
}

func WithReference_Serialization(serialization string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Serialization = serialization
	}
}

func WithReference_ProviderBy(providedBy string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.ProvidedBy = providedBy
	}
}

func WithReference_Async(async bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Async = async
	}
}

func WithReference_Params(params map[string]string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Params = params
	}
}

func WithReference_Generic(generic string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Generic = generic
	}
}

func WithReference_Sticky(sticky bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Sticky = sticky
	}
}

func WithReference_RequestTimeout(timeout string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.RequestTimeout = timeout
	}
}

func WithReference_Force(force bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.ForceTag = force
	}
}

func WithReference_TracingKey(tracingKey string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.TracingKey = tracingKey
	}
}

func WithReference_MeshProviderPort(port int) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.MeshProviderPort = port
	}
}
