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

package config

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type mockGracefulShutdownSetterFilter struct {
	settings map[string]any
}

func newMockGracefulShutdownSetterFilter() *mockGracefulShutdownSetterFilter {
	return &mockGracefulShutdownSetterFilter{
		settings: make(map[string]any),
	}
}

func (m *mockGracefulShutdownSetterFilter) Invoke(context.Context, base.Invoker, base.Invocation) result.Result {
	return nil
}

func (m *mockGracefulShutdownSetterFilter) OnResponse(context.Context, result.Result, base.Invoker, base.Invocation) result.Result {
	return nil
}

func (m *mockGracefulShutdownSetterFilter) Set(name string, conf any) {
	m.settings[name] = conf
}

func TestGracefulShutdownInitSetsShutdownConfigOnFilters(t *testing.T) {
	consumerFilter := newMockGracefulShutdownSetterFilter()
	providerFilter := newMockGracefulShutdownSetterFilter()
	extension.SetFilter(constant.GracefulShutdownConsumerFilterKey, func() filter.Filter {
		return consumerFilter
	})
	extension.SetFilter(constant.GracefulShutdownProviderFilterKey, func() filter.Filter {
		return providerFilter
	})
	t.Cleanup(func() {
		extension.UnregisterFilter(constant.GracefulShutdownConsumerFilterKey)
		extension.UnregisterFilter(constant.GracefulShutdownProviderFilterKey)
	})

	shutdownConfig := NewShutDownConfigBuilder().
		SetInternalSignal(false).
		Build()
	SetRootConfig(*NewRootConfigBuilder().
		SetShutdown(shutdownConfig).
		Build())

	gracefulShutdownInit()

	consumerShutdownConfig, ok := consumerFilter.settings[constant.GracefulShutdownFilterShutdownConfig]
	require.True(t, ok)
	assert.Same(t, shutdownConfig, consumerShutdownConfig)
	providerShutdownConfig, ok := providerFilter.settings[constant.GracefulShutdownFilterShutdownConfig]
	require.True(t, ok)
	assert.Same(t, shutdownConfig, providerShutdownConfig)
}

//
//import (
//	"dubbo.apache.org/dubbo-go/v3/config"
//	"dubbo.apache.org/dubbo-go/v3/config/consumer"
//	protocol2 "dubbo.apache.org/dubbo-go/v3/config/protocol"
//	"dubbo.apache.org/dubbo-go/v3/config/provider"
//	"dubbo.apache.org/dubbo-go/v3/config/reference"
//	"testing"
//)
//
//import (
//	"dubbo.apache.org/dubbo-go/v3/common/constant"
//	"dubbo.apache.org/dubbo-go/v3/common/extension"
//	"dubbo.apache.org/dubbo-go/v3/filter"
//	"dubbo.apache.org/dubbo-go/v3/protocol"
//)
//
//func TestGracefulShutdownInit(t *testing.T) {
//	extension.SetFilter(constant.GracefulShutdownConsumerFilterKey, func() filter.Filter {
//		return &config.mockGracefulShutdownFilter{}
//	})
//	extension.SetFilter(constant.GracefulShutdownProviderFilterKey, func() filter.Filter {
//		return &config.mockGracefulShutdownFilter{}
//	})
//	GracefulShutdownInit()
//}
//
//func TestBeforeShutdown(t *testing.T) {
//	extension.SetProtocol("registry", func() protocol.Protocol {
//		return &config.mockRegistryProtocol{}
//	})
//	extension.SetProtocol(constant.DUBBO, func() protocol.Protocol {
//		return &config.mockRegistryProtocol{}
//	})
//
//	extension.SetProtocol("mock", func() protocol.Protocol {
//		return &config.mockRegistryProtocol{}
//	})
//
//	consumerReferences := map[string]*reference.ReferenceConfig{}
//	consumerReferences[constant.DUBBO] = &reference.ReferenceConfig{
//		Protocol: constant.DUBBO,
//	}
//
//	// without configuration
//	config.consumerConfig = nil
//	config.referenceConfig = nil
//	BeforeShutdown()
//
//	config.consumerConfig = &consumer.ShutdownConfig{
//		References: consumerReferences,
//		ShutdownConfig: &ShutdownConfig{
//			Timeout:     "1",
//			StepTimeout: "1s",
//		},
//	}
//
//	providerProtocols := map[string]*protocol2.ProtocolConfig{}
//	providerProtocols[constant.DUBBO] = &protocol2.ProtocolConfig{
//		Name: constant.DUBBO,
//	}
//
//	providerProtocols["mock"] = &protocol2.ProtocolConfig{
//		Name: "mock",
//	}
//
//	config.referenceConfig = &provider.ProviderConfig{
//		ShutdownConfig: &ShutdownConfig{
//			Timeout:     "1",
//			StepTimeout: "1s",
//		},
//		Protocols: providerProtocols,
//	}
//	// test destroy protocol
//	BeforeShutdown()
//
//	config.referenceConfig = &provider.ProviderConfig{
//		ShutdownConfig: &ShutdownConfig{
//			Timeout:     "1",
//			StepTimeout: "-1s",
//		},
//		Protocols: providerProtocols,
//	}
//
//	config.consumerConfig = &consumer.ShutdownConfig{
//		References: consumerReferences,
//		ShutdownConfig: &ShutdownConfig{
//			Timeout:     "1",
//			StepTimeout: "-1s",
//		},
//	}
//
//	// test ignore steps
//	BeforeShutdown()
//}
