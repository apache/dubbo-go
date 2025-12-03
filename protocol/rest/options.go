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

package rest

import (
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"

	restconfig "dubbo.apache.org/dubbo-go/v3/protocol/rest/config"
)

// ServiceOption customizes RestServiceConfig.
type ServiceOption func(*restconfig.RestServiceConfig)

// MethodOption customizes RestMethodConfig.
type MethodOption func(*restconfig.RestMethodConfig)

// NewServiceConfig constructs a RestServiceConfig via functional options.
func NewServiceConfig(opts ...ServiceOption) *restconfig.RestServiceConfig {
	cfg := &restconfig.RestServiceConfig{
		RestMethodConfigs: make([]*restconfig.RestMethodConfig, 0),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// NewMethodConfig builds a RestMethodConfig via functional options.
func NewMethodConfig(name string, opts ...MethodOption) *restconfig.RestMethodConfig {
	method := &restconfig.RestMethodConfig{
		MethodName: name,
		Body:       -1,
	}
	for _, opt := range opts {
		opt(method)
	}
	return method
}

// ApplyConsumerServiceConfig registers the given service config for consumer side.
func ApplyConsumerServiceConfig(id string, cfg *restconfig.RestServiceConfig) {
	if cfg == nil || id == "" {
		return
	}
	finalizeServiceConfig(cfg)
	restconfig.UpsertRestConsumerServiceConfig(id, cfg)
}

// ApplyProviderServiceConfig registers the given service config for provider side.
func ApplyProviderServiceConfig(id string, cfg *restconfig.RestServiceConfig) {
	if cfg == nil || id == "" {
		return
	}
	finalizeServiceConfig(cfg)
	restconfig.UpsertRestProviderServiceConfig(id, cfg)
}

// WithServiceInterface sets the Java-style interface for the rest service.
func WithServiceInterface(interfaceName string) ServiceOption {
	return func(cfg *restconfig.RestServiceConfig) {
		cfg.InterfaceName = interfaceName
	}
}

// WithServiceURL sets the base url.
func WithServiceURL(url string) ServiceOption {
	return func(cfg *restconfig.RestServiceConfig) {
		cfg.URL = url
	}
}

// WithServicePath sets the base path that will prefix all method paths.
func WithServicePath(path string) ServiceOption {
	return func(cfg *restconfig.RestServiceConfig) {
		cfg.Path = path
	}
}

// WithServiceProduces sets the default produces header for all methods.
func WithServiceProduces(produces string) ServiceOption {
	return func(cfg *restconfig.RestServiceConfig) {
		cfg.Produces = produces
	}
}

// WithServiceConsumes sets the default consumes header for all methods.
func WithServiceConsumes(consumes string) ServiceOption {
	return func(cfg *restconfig.RestServiceConfig) {
		cfg.Consumes = consumes
	}
}

// WithServiceClient specifies the rest client implementation (default: resty).
func WithServiceClient(client string) ServiceOption {
	return func(cfg *restconfig.RestServiceConfig) {
		cfg.Client = client
	}
}

// WithServiceServer specifies underlying rest server implementation (default: go-restful).
func WithServiceServer(server string) ServiceOption {
	return func(cfg *restconfig.RestServiceConfig) {
		cfg.Server = server
	}
}

// WithServiceMethod appends a rest method definition to the service.
func WithServiceMethod(name string, opts ...MethodOption) ServiceOption {
	return func(cfg *restconfig.RestServiceConfig) {
		method := NewMethodConfig(name, opts...)
		cfg.RestMethodConfigs = append(cfg.RestMethodConfigs, method)
	}
}

// WithMethodPath configures the method path (can be relative to service path).
func WithMethodPath(path string) MethodOption {
	return func(method *restconfig.RestMethodConfig) {
		method.Path = path
	}
}

// WithMethodURL configures an absolute URL for the method.
func WithMethodURL(url string) MethodOption {
	return func(method *restconfig.RestMethodConfig) {
		method.URL = url
	}
}

// WithMethodProduces sets the produces header for the method.
func WithMethodProduces(produces string) MethodOption {
	return func(method *restconfig.RestMethodConfig) {
		method.Produces = produces
	}
}

// WithMethodConsumes sets the consumes header for the method.
func WithMethodConsumes(consumes string) MethodOption {
	return func(method *restconfig.RestMethodConfig) {
		method.Consumes = consumes
	}
}

// WithMethodHTTPMethod sets HTTP verb for the method (GET/POST/...).
func WithMethodHTTPMethod(methodType string) MethodOption {
	return func(method *restconfig.RestMethodConfig) {
		method.MethodType = methodType
	}
}

// WithMethodBodyIndex sets the index of argument bound to request body.
func WithMethodBodyIndex(index int) MethodOption {
	return func(method *restconfig.RestMethodConfig) {
		method.Body = index
	}
}

// WithMethodPathParam binds a path param name to an argument index.
func WithMethodPathParam(index int, name string) MethodOption {
	return func(method *restconfig.RestMethodConfig) {
		if method.PathParamsMap == nil {
			method.PathParamsMap = make(map[int]string)
		}
		method.PathParamsMap[index] = name
	}
}

// WithMethodQueryParam binds a query param name to an argument index.
func WithMethodQueryParam(index int, name string) MethodOption {
	return func(method *restconfig.RestMethodConfig) {
		if method.QueryParamsMap == nil {
			method.QueryParamsMap = make(map[int]string)
		}
		method.QueryParamsMap[index] = name
	}
}

// WithMethodHeader binds a header name to an argument index.
func WithMethodHeader(index int, name string) MethodOption {
	return func(method *restconfig.RestMethodConfig) {
		if method.HeadersMap == nil {
			method.HeadersMap = make(map[int]string)
		}
		method.HeadersMap[index] = name
	}
}

// finalizeServiceConfig normalizes a RestServiceConfig and populates its method map.
func finalizeServiceConfig(cfg *restconfig.RestServiceConfig) {
	if cfg == nil {
		return
	}
	if cfg.RestMethodConfigsMap == nil {
		cfg.RestMethodConfigsMap = make(map[string]*restconfig.RestMethodConfig, len(cfg.RestMethodConfigs))
	}
	for _, method := range cfg.RestMethodConfigs {
		if method == nil || method.MethodName == "" {
			continue
		}
		if method.InterfaceName == "" {
			method.InterfaceName = cfg.InterfaceName
		}
		if cfg.Path != "" && method.Path != "" {
			method.Path = joinPath(cfg.Path, method.Path)
		} else if method.Path == "" {
			method.Path = cfg.Path
		}
		if existing, ok := cfg.RestMethodConfigsMap[method.MethodName]; ok {
			// Warn when duplicate method name is encountered to avoid silent overwrite
			logger.Warnf("finalizeServiceConfig: duplicate method name %q for service %q; existing method %+v will be overwritten by %+v",
				method.MethodName, cfg.InterfaceName, existing, method)
		}
		cfg.RestMethodConfigsMap[method.MethodName] = method
	}
}

func joinPath(base, sub string) string {
	base = strings.TrimSpace(base)
	sub = strings.TrimSpace(sub)
	if base == "" {
		return sub
	}
	if sub == "" {
		return base
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(sub, "/")
}
