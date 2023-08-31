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

package client

import (
	"strconv"
)

import (
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// todo: need to be consistent with MethodConfig
type CallOptions struct {
	RequestTimeout string
	Retries        string
}

type CallOption func(*CallOptions)

func newDefaultCallOptions() *CallOptions {
	return &CallOptions{
		RequestTimeout: "",
		Retries:        "",
	}
}

func WithCallRequestTimeout(timeout string) CallOption {
	return func(opts *CallOptions) {
		opts.RequestTimeout = timeout
	}
}

func WithCallRetries(retries string) CallOption {
	return func(opts *CallOptions) {
		opts.Retries = retries
	}
}

// ----------ReferenceOption----------

// For ReferenceOption that needs to check whether configuration field is empty(eg. WithCheck), it means this
// ReferenceOption maybe used by ConsumerConfig to act as default value.

func WithCheck(check bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		if cfg.Check == nil {
			cfg.Check = &check
		}
	}
}

func WithURL(url string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.URL = url
	}
}

func WithFilter(filter string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		if cfg.Filter == "" {
			cfg.Filter = filter
		}
	}
}

func WithProtocol(protocol string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		if cfg.Protocol == "" {
			cfg.Protocol = protocol
		}
	}
}

func WithRegistryIDs(registryIDs []string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		if len(registryIDs) <= 0 {
			cfg.RegistryIDs = registryIDs
		}
	}
}

func WithCluster(cluster string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Cluster = cluster
	}
}

func WithLoadBalance(loadBalance string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Loadbalance = loadBalance
	}
}

func WithRetries(retries int) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Retries = strconv.Itoa(retries)
	}
}

func WithGroup(group string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Group = group
	}
}

func WithVersion(version string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Version = version
	}
}

func WithSerialization(serialization string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Serialization = serialization
	}
}

func WithProviderBy(providedBy string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.ProvidedBy = providedBy
	}
}

func WithAsync(async bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Async = async
	}
}

func WithParams(params map[string]string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Params = params
	}
}

func WithGeneric(generic string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Generic = generic
	}
}

func WithSticky(sticky bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.Sticky = sticky
	}
}

func WithRequestTimeout(timeout string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.RequestTimeout = timeout
	}
}

func WithForce(force bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.ForceTag = force
	}
}

func WithTracingKey(tracingKey string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		if cfg.TracingKey == "" {
			cfg.TracingKey = tracingKey
		}
	}
}

func WithMeshProviderPort(port int) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.MeshProviderPort = port
	}
}

// ----------From ApplicationConfig----------

func WithApplication(application *commonCfg.ApplicationConfig) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.application = application
	}
}

func WithApplicationOrganization(organization string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.applicationOrganization = organization
	}
}

func WithApplicationName(name string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.applicationName = name
	}
}

func WithApplicationModule(module string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.applicationModule = module
	}
}

func WithApplicationGroup(group string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.applicationGroup = group
	}
}

func WithApplicationVersion(version string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.applicationVersion = version
	}
}

func WithApplicationOwner(owner string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.applicationOwner = owner
	}
}

func WithApplicationEnvironment(environment string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.applicationEnvironment = environment
	}
}

func WithApplicationMetadataType(typ string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.applicationMetadataType = typ
	}
}

func WithApplicationTag(tag string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.applicationTag = tag
	}
}

// ----------From ConsumerConfig----------

func WithMeshEnabled(meshEnabled bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.meshEnabled = meshEnabled
	}
}

func WithAdaptiveService(adaptiveService bool) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.adaptiveService = adaptiveService
	}
}

func WithProxyFactory(proxyFactory string) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.proxyFactory = proxyFactory
	}
}

// ----------From RegistryConfig----------

func WithRegistries(registries map[string]*registry.RegistryConfig) ReferenceOption {
	return func(cfg *ReferenceConfig) {
		cfg.registries = registries
	}
}
