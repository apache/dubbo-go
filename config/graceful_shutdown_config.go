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
	defaultTimeout                     = 60 * time.Second
	defaultStepTimeout                 = 3 * time.Second
	defaultConsumerUpdateWaitTime      = 3 * time.Second
	defaultOfflineRequestWindowTimeout = 3 * time.Second
)

// ShutdownConfig is used as configuration for graceful shutdown
type ShutdownConfig struct {
	/*
	 * Total timeout. Even though we don't release all resources,
	 * the applicationConfig will shutdown if the costing time is over this configuration. The unit is ms.
	 * default value is 60 * 1000 ms = 1 minutes
	 * In general, it should be bigger than 3 * StepTimeout.
	 */
	//总超时时间：即使资源未完全释放，超过此时间应用也会强制关闭
	Timeout string `default:"60s" yaml:"timeout" json:"timeout,omitempty" property:"timeout"`
	/*
	 * the timeout on each step. You should evaluate the response time of request
	 * and the time that client noticed that server shutdown.
	 * For example, if your client will received the notification within 10s when you start to close server,
	 * and the 99.9% requests will return response in 2s, so the StepTimeout will be bigger than(10+2) * 1000ms,
	 * maybe (10 + 2*3) * 1000ms is a good choice.
	 */
	//步骤超时时间：评估请求响应时间和客户端感知服务器关闭的时间
	StepTimeout string `default:"3s" yaml:"step-timeout" json:"step.timeout,omitempty" property:"step.timeout"`

	/*
	 * ConsumerUpdateWaitTime means when provider is shutting down, after the unregister, time to wait for client to
	 * update invokers. During this time, incoming invocation can be treated normally.
	 */
	//  消费者更新等待时间：提供者关闭时，等待客户端更新调用者的时间
	ConsumerUpdateWaitTime string `default:"3s" yaml:"consumer-update-wait-time" json:"consumerUpdate.waitTIme,omitempty" property:"consumerUpdate.waitTIme"`
	// when we try to shutdown the applicationConfig, we will reject the new requests. In most cases, you don't need to configure this.
	// 请求拒绝处理器：配置当拒绝新请求时使用的处理器
	RejectRequestHandler string `yaml:"reject-handler" json:"reject-handler,omitempty" property:"reject_handler"`
	// internal listen kill signal，the default is true.
	// 内部信号监听：是否监听终止信号，默认true
	InternalSignal *bool `default:"true" yaml:"internal-signal" json:"internal.signal,omitempty" property:"internal.signal"`
	// offline request window length
	// 离线请求窗口超时时间
	OfflineRequestWindowTimeout string `yaml:"offline-request-window-timeout" json:"offlineRequestWindowTimeout,omitempty" property:"offlineRequestWindowTimeout"`
	// true -> new request will be rejected.
	// 拒绝新请求标志：true表示拒绝新请求
	RejectRequest atomic.Bool
	// active invocation
	// 活跃调用计数
	ConsumerActiveCount atomic.Int32
	ProviderActiveCount atomic.Int32

	// provider last received request timestamp
	// 提供者最后接收请求的时间戳
	ProviderLastReceivedRequestTime atomic.Time
}

// Prefix dubbo.shutdown
// 返回配置前缀
func (config *ShutdownConfig) Prefix() string {
	return constant.ShutdownConfigPrefix
}

// GetTimeout 获取超时时间
func (config *ShutdownConfig) GetTimeout() time.Duration {
	result, err := time.ParseDuration(config.Timeout)
	if err != nil {
		logger.Errorf("The Timeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			config.Timeout, defaultTimeout.String(), err)
		return defaultTimeout
	}
	return result
}

// GetStepTimeout 获取步骤超时时间
func (config *ShutdownConfig) GetStepTimeout() time.Duration {
	result, err := time.ParseDuration(config.StepTimeout)
	if err != nil {
		logger.Errorf("The StepTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			config.StepTimeout, defaultStepTimeout.String(), err)
		return defaultStepTimeout
	}
	return result
}

// GetOfflineRequestWindowTimeout 获取离线请求窗口超时时间
func (config *ShutdownConfig) GetOfflineRequestWindowTimeout() time.Duration {
	result, err := time.ParseDuration(config.OfflineRequestWindowTimeout)
	if err != nil {
		logger.Errorf("The OfflineRequestWindowTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			config.OfflineRequestWindowTimeout, defaultOfflineRequestWindowTimeout.String(), err)
		return defaultOfflineRequestWindowTimeout
	}
	return result
}

// GetConsumerUpdateWaitTime 获取消费者更新等待时间
func (config *ShutdownConfig) GetConsumerUpdateWaitTime() time.Duration {
	result, err := time.ParseDuration(config.ConsumerUpdateWaitTime)
	if err != nil {
		logger.Errorf("The ConsumerUpdateTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			config.ConsumerActiveCount.Load(), defaultConsumerUpdateWaitTime.String(), err)
		return defaultConsumerUpdateWaitTime
	}
	return result
}

// GetInternalSignal 获取内部信号监听标志
func (config *ShutdownConfig) GetInternalSignal() bool {
	if config.InternalSignal == nil {
		return false
	}
	return *config.InternalSignal
}

// Init 初始化配置
func (config *ShutdownConfig) Init() error {
	return defaults.Set(config)
}

// ShutdownConfigBuilder 关闭配置构建器
type ShutdownConfigBuilder struct {
	shutdownConfig *ShutdownConfig
}

// NewShutDownConfigBuilder 创建关闭配置构建器
func NewShutDownConfigBuilder() *ShutdownConfigBuilder {
	return &ShutdownConfigBuilder{shutdownConfig: &ShutdownConfig{}}
}

// SetTimeout 设置超时时间
func (scb *ShutdownConfigBuilder) SetTimeout(timeout string) *ShutdownConfigBuilder {
	scb.shutdownConfig.Timeout = timeout
	return scb
}

// SetStepTimeout 设置步骤超时时间
func (scb *ShutdownConfigBuilder) SetStepTimeout(stepTimeout string) *ShutdownConfigBuilder {
	scb.shutdownConfig.StepTimeout = stepTimeout
	return scb
}

// SetRejectRequestHandler 设置拒绝请求处理器
func (scb *ShutdownConfigBuilder) SetRejectRequestHandler(rejectRequestHandler string) *ShutdownConfigBuilder {
	scb.shutdownConfig.RejectRequestHandler = rejectRequestHandler
	return scb
}

// SetRejectRequest 设置拒绝新请求标志
func (scb *ShutdownConfigBuilder) SetRejectRequest(rejectRequest bool) *ShutdownConfigBuilder {
	scb.shutdownConfig.RejectRequest.Store(rejectRequest)
	return scb
}

// SetInternalSignal 设置内部信号监听标志
func (scb *ShutdownConfigBuilder) SetInternalSignal(internalSignal bool) *ShutdownConfigBuilder {
	scb.shutdownConfig.InternalSignal = &internalSignal
	return scb
}

// SetOfflineRequestWindowTimeout 设置离线请求窗口超时时间
func (scb *ShutdownConfigBuilder) Build() *ShutdownConfig {
	defaults.MustSet(scb.shutdownConfig)
	return scb.shutdownConfig
}

// SetOfflineRequestWindowTimeout 设置离线请求窗口超时时间
func (scb *ShutdownConfigBuilder) SetOfflineRequestWindowTimeout(offlineRequestWindowTimeout string) *ShutdownConfigBuilder {
	scb.shutdownConfig.OfflineRequestWindowTimeout = offlineRequestWindowTimeout
	return scb
}
