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
	"time"
)

import (
	"github.com/creasty/defaults"

	"github.com/dubbogo/gost/log/logger"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

const (
	defaultTimeout                = 60 * time.Second
	defaultStepTimeout            = 3 * time.Second
	defaultConsumerUpdateWaitTime = 3 * time.Second
)

// ShutdownConfig is used as configuration for graceful shutdown
type ShutdownConfig struct {
	/*
	 * Total timeout. Even though we don't release all resources,
	 * the applicationConfig will shutdown if the costing time is over this configuration. The unit is ms.
	 * default value is 60 * 1000 ms = 1 minutes
	 * In general, it should be bigger than 3 * StepTimeout.
	 */
	Timeout string `default:"60s" yaml:"timeout" json:"timeout,omitempty" property:"timeout"`
	/*
	 * the timeout on each step. You should evaluate the response time of request
	 * and the time that client noticed that server shutdown.
	 * For example, if your client will received the notification within 10s when you start to close server,
	 * and the 99.9% requests will return response in 2s, so the StepTimeout will be bigger than(10+2) * 1000ms,
	 * maybe (10 + 2*3) * 1000ms is a good choice.
	 */
	StepTimeout string `default:"3s" yaml:"step-timeout" json:"step.timeout,omitempty" property:"step.timeout"`

	/*
	 * ConsumerUpdateWaitTime means when provider is shutting down, after the unregister, time to wait for client to
	 * update invokers. During this time, incoming invocation can be treated normally.
	 */
	ConsumerUpdateWaitTime string `default:"3s" yaml:"consumer-update-wait-time" json:"consumerUpdate.waitTIme,omitempty" property:"consumerUpdate.waitTIme"`
	// when we try to shutdown the applicationConfig, we will reject the new requests. In most cases, you don't need to configure this.
	RejectRequestHandler string `yaml:"reject-handler" json:"reject-handler,omitempty" property:"reject_handler"`
	// internal listen kill signal，the default is true.
	InternalSignal bool `default:"true" yaml:"internal-signal" json:"internal.signal,omitempty" property:"internal.signal"`

	// true -> new request will be rejected.
	RejectRequest atomic.Bool
	// active invocation
	ConsumerActiveCount atomic.Int32
	ProviderActiveCount atomic.Int32
}

// Prefix dubbo.shutdown
func (config *ShutdownConfig) Prefix() string {
	return constant.ShutdownConfigPrefix
}

func (config *ShutdownConfig) GetTimeout() time.Duration {
	result, err := time.ParseDuration(config.Timeout)
	if err != nil {
		logger.Errorf("The Timeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			config.Timeout, defaultTimeout.String(), err)
		return defaultTimeout
	}
	return result
}

func (config *ShutdownConfig) GetStepTimeout() time.Duration {
	result, err := time.ParseDuration(config.StepTimeout)
	if err != nil {
		logger.Errorf("The StepTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			config.StepTimeout, defaultStepTimeout.String(), err)
		return defaultStepTimeout
	}
	return result
}

func (config *ShutdownConfig) GetConsumerUpdateWaitTime() time.Duration {
	result, err := time.ParseDuration(config.ConsumerUpdateWaitTime)
	if err != nil {
		logger.Errorf("The ConsumerUpdateTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			config.ConsumerActiveCount.Load(), defaultConsumerUpdateWaitTime.String(), err)
		return defaultConsumerUpdateWaitTime
	}
	return result
}

func (config *ShutdownConfig) Init() error {
	return defaults.Set(config)
}

type ShutdownConfigBuilder struct {
	shutdownConfig *ShutdownConfig
}

func NewShutDownConfigBuilder() *ShutdownConfigBuilder {
	return &ShutdownConfigBuilder{shutdownConfig: &ShutdownConfig{}}
}

func (scb *ShutdownConfigBuilder) SetTimeout(timeout string) *ShutdownConfigBuilder {
	scb.shutdownConfig.Timeout = timeout
	return scb
}

func (scb *ShutdownConfigBuilder) SetStepTimeout(stepTimeout string) *ShutdownConfigBuilder {
	scb.shutdownConfig.StepTimeout = stepTimeout
	return scb
}

func (scb *ShutdownConfigBuilder) SetRejectRequestHandler(rejectRequestHandler string) *ShutdownConfigBuilder {
	scb.shutdownConfig.RejectRequestHandler = rejectRequestHandler
	return scb
}

func (scb *ShutdownConfigBuilder) SetRejectRequest(rejectRequest bool) *ShutdownConfigBuilder {
	scb.shutdownConfig.RejectRequest.Store(rejectRequest)
	return scb
}

func (scb *ShutdownConfigBuilder) SetInternalSignal(internalSignal bool) *ShutdownConfigBuilder {
	scb.shutdownConfig.InternalSignal = internalSignal
	return scb
}

func (scb *ShutdownConfigBuilder) Build() *ShutdownConfig {
	defaults.Set(scb)
	return scb.shutdownConfig
}
