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
	 * NotifyTimeout means the timeout budget for actively notifying long-connection consumers
	 * during graceful shutdown. It only controls the notify step and should not be coupled to
	 * request draining timeouts.
	 */
	NotifyTimeout string `default:"5s" yaml:"notify-timeout" json:"notify.timeout,omitempty" property:"notify.timeout"`

	/*
	 * ConsumerUpdateWaitTime means when provider is shutting down, after the unregister, time to wait for client to
	 * update invokers. During this time, incoming invocation can be treated normally.
	 */
	ConsumerUpdateWaitTime string `default:"3s" yaml:"consumer-update-wait-time" json:"consumerUpdate.waitTIme,omitempty" property:"consumerUpdate.waitTIme"`
	// when we try to shutdown the applicationConfig, we will reject the new requests. In most cases, you don't need to configure this.
	RejectRequestHandler string `yaml:"reject-handler" json:"reject-handler,omitempty" property:"reject_handler"`
	// internal listen kill signal，the default is true.
	InternalSignal *bool `default:"true" yaml:"internal-signal" json:"internal.signal,omitempty" property:"internal.signal"`
	// offline request window length
	OfflineRequestWindowTimeout string `default:"3s" yaml:"offline-request-window-timeout" json:"offlineRequestWindowTimeout,omitempty" property:"offlineRequestWindowTimeout"`
	// true -> new request will be rejected.
	RejectRequest atomic.Bool
	// active invocation
	ConsumerActiveCount atomic.Int32
	ProviderActiveCount atomic.Int32

	// provider last received request timestamp
	ProviderLastReceivedRequestTime atomic.Time

	Closing atomic.Bool

	// ClosingInvokerExpireTime controls how long the consumer keeps an invoker
	// marked as closing after receiving an active/passive closing signal.
	ClosingInvokerExpireTime string `default:"30s" yaml:"closing-invoker-expire-time" json:"closingInvokerExpireTime,omitempty" property:"closingInvokerExpireTime"`
}

func DefaultShutdownConfig() *ShutdownConfig {
	cfg := &ShutdownConfig{}
	defaults.MustSet(cfg)

	return cfg
}

// Prefix dubbo.shutdown
func (c *ShutdownConfig) Prefix() string {
	return constant.ShutdownConfigPrefix
}

func (c *ShutdownConfig) GetTimeout() time.Duration {
	result, err := time.ParseDuration(c.Timeout)
	if err != nil {
		logger.Errorf("The Timeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			c.Timeout, constant.DefaultShutdownConfigTimeout.String(), err)
		return constant.DefaultShutdownConfigTimeout
	}
	return result
}

func (c *ShutdownConfig) GetStepTimeout() time.Duration {
	result, err := time.ParseDuration(c.StepTimeout)
	if err != nil {
		logger.Errorf("The StepTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			c.StepTimeout, constant.DefaultShutdownConfigStepTimeout.String(), err)
		return constant.DefaultShutdownConfigStepTimeout
	}
	return result
}

func (c *ShutdownConfig) GetNotifyTimeout() time.Duration {
	result, err := time.ParseDuration(c.NotifyTimeout)
	if err != nil {
		logger.Errorf("The NotifyTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			c.NotifyTimeout, constant.DefaultShutdownConfigNotifyTimeout.String(), err)
		return constant.DefaultShutdownConfigNotifyTimeout
	}
	return result
}

func (c *ShutdownConfig) GetOfflineRequestWindowTimeout() time.Duration {
	result, err := time.ParseDuration(c.OfflineRequestWindowTimeout)
	if err != nil {
		logger.Errorf("The OfflineRequestWindowTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			c.OfflineRequestWindowTimeout, constant.DefaultShutdownConfigOfflineRequestWindowTimeout.String(), err)
		return constant.DefaultShutdownConfigOfflineRequestWindowTimeout
	}
	return result
}

func (c *ShutdownConfig) GetConsumerUpdateWaitTime() time.Duration {
	result, err := time.ParseDuration(c.ConsumerUpdateWaitTime)
	if err != nil {
		logger.Errorf("The ConsumerUpdateTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			c.ConsumerActiveCount.Load(), constant.DefaultShutdownConfigConsumerUpdateWaitTime.String(), err)
		return constant.DefaultShutdownConfigConsumerUpdateWaitTime
	}
	return result
}

func (c *ShutdownConfig) GetInternalSignal() bool {
	if c.InternalSignal == nil {
		return false
	}
	return *c.InternalSignal
}

func (c *ShutdownConfig) Init() error {
	return defaults.Set(c)
}

// Clone a new ShutdownConfig
func (c *ShutdownConfig) Clone() *ShutdownConfig {
	if c == nil {
		return nil
	}

	var newInternalSignal *bool
	if c.InternalSignal != nil {
		newInternalSignal = new(bool)
		*newInternalSignal = *c.InternalSignal
	}

	newShutdownConfig := &ShutdownConfig{
		Timeout:                     c.Timeout,
		StepTimeout:                 c.StepTimeout,
		NotifyTimeout:               c.NotifyTimeout,
		ConsumerUpdateWaitTime:      c.ConsumerUpdateWaitTime,
		RejectRequestHandler:        c.RejectRequestHandler,
		InternalSignal:              newInternalSignal,
		OfflineRequestWindowTimeout: c.OfflineRequestWindowTimeout,
		ClosingInvokerExpireTime:    c.ClosingInvokerExpireTime,
	}

	newShutdownConfig.RejectRequest.Store(c.RejectRequest.Load())
	newShutdownConfig.ConsumerActiveCount.Store(c.ConsumerActiveCount.Load())
	newShutdownConfig.ProviderActiveCount.Store(c.ProviderActiveCount.Load())
	newShutdownConfig.ProviderLastReceivedRequestTime.Store(c.ProviderLastReceivedRequestTime.Load())
	newShutdownConfig.Closing.Store(c.Closing.Load())

	return newShutdownConfig
}
