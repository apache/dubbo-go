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
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

func TestGracefulShutdownInit(t *testing.T) {
	extension.SetFilter(constant.GracefulShutdownConsumerFilterKey, func() filter.Filter {
		return &mockGracefulShutdownFilter{}
	})
	extension.SetFilter(constant.GracefulShutdownProviderFilterKey, func() filter.Filter {
		return &mockGracefulShutdownFilter{}
	})
	GracefulShutdownInit()
}

func TestBeforeShutdown(t *testing.T) {
	extension.SetProtocol("registry", func() protocol.Protocol {
		return &mockRegistryProtocol{}
	})
	extension.SetProtocol(constant.DUBBO, func() protocol.Protocol {
		return &mockRegistryProtocol{}
	})

	extension.SetProtocol("mock", func() protocol.Protocol {
		return &mockRegistryProtocol{}
	})

	consumerReferences := map[string]*ReferenceConfig{}
	consumerReferences[constant.DUBBO] = &ReferenceConfig{
		Protocol: constant.DUBBO,
	}

	// without configuration
	consumerConfig = nil
	providerConfig = nil
	BeforeShutdown()

	consumerConfig = &ConsumerConfig{
		References: consumerReferences,
		ShutdownConfig: &ShutdownConfig{
			Timeout:     "1",
			StepTimeout: "1s",
		},
	}

	providerProtocols := map[string]*ProtocolConfig{}
	providerProtocols[constant.DUBBO] = &ProtocolConfig{
		Name: constant.DUBBO,
	}

	providerProtocols["mock"] = &ProtocolConfig{
		Name: "mock",
	}

	providerConfig = &ProviderConfig{
		ShutdownConfig: &ShutdownConfig{
			Timeout:     "1",
			StepTimeout: "1s",
		},
		Protocols: providerProtocols,
	}
	// test destroy protocol
	BeforeShutdown()

	providerConfig = &ProviderConfig{
		ShutdownConfig: &ShutdownConfig{
			Timeout:     "1",
			StepTimeout: "-1s",
		},
		Protocols: providerProtocols,
	}

	consumerConfig = &ConsumerConfig{
		References: consumerReferences,
		ShutdownConfig: &ShutdownConfig{
			Timeout:     "1",
			StepTimeout: "-1s",
		},
	}

	// test ignore steps
	BeforeShutdown()
}
