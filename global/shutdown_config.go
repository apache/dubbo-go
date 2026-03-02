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

	"github.com/creasty/defaults"

	"go.uber.org/atomic"
)

// ShutdownConfig is used as configuration for graceful shutdown
type ShutdownConfig struct {
	/*
	 * Total timeout. Even though we don't release all resources,
	 * the applicationConfig will shutdown if the costing time is over this configuration. The unit is ms.
	 * default value is 60 * 1000 ms = 1 minutes
	 * In general, it should be bigger than 3 * StepTimeout.
	 * 完全暂停。即使我们没有释放所有资源，但如果成本时间超过这个配置，applicationConfig 也会关闭。
	 * 单位是毫秒。默认值是60 1000毫秒 = 1分钟。一般来说，它应该大于3倍StepTimeout超时。
	 */
	Timeout string `default:"60s" yaml:"timeout" json:"timeout,omitempty" property:"timeout"`
	/*
	 * the timeout on each step. You should evaluate the response time of request
	 * and the time that client noticed that server shutdown.
	 * For example, if your client will received the notification within 10s when you start to close server,
	 * and the 99.9% requests will return response in 2s, so the StepTimeout will be bigger than(10+2) * 1000ms,
	 * maybe (10 + 2*3) * 1000ms is a good choice.
	 *	每一步的超时。你应该评估请求的响应时间以及客户端注意到服务器关闭的时间。
	 *	例如，如果你的客户端在你开始关闭服务器时10秒内收到通知，而99.9%的请求会在2秒内返回，
	 *	所以StepTimeout会大于（10+2）1000毫秒，也许（10 + 23）1000毫秒是个不错的选择。
	 */
	StepTimeout string `default:"3s" yaml:"step-timeout" json:"step.timeout,omitempty" property:"step.timeout"`

	/*
	 * ConsumerUpdateWaitTime means when provider is shutting down, after the unregister, time to wait for client to
	 * update invokers. During this time, incoming invocation can be treated normally.
	 * ConsumerUpdateWaitTime 指的是当提供者关闭时，在取消注册后等待客户端更新调用器的等待时间。
	 * 在此期间，收到的召唤可以正常处理。
	 */
	ConsumerUpdateWaitTime string `default:"3s" yaml:"consumer-update-wait-time" json:"consumerUpdate.waitTIme,omitempty" property:"consumerUpdate.waitTIme"`
	// when we try to shutdown the applicationConfig, we will reject the new requests. In most cases, you don't need to configure this.
	// 当我们尝试关闭应用程序时，我们会拒绝新请求。在大多数情况下，您不需要配置此选项。
	RejectRequestHandler string `yaml:"reject-handler" json:"reject-handler,omitempty" property:"reject_handler"`
	// internal listen kill signal，the default is true.
	// 内部监听停止信号，默认值为真
	InternalSignal *bool `default:"true" yaml:"internal-signal" json:"internal.signal,omitempty" property:"internal.signal"`
	// offline request window length
	// 下线请求窗口大小
	OfflineRequestWindowTimeout string `yaml:"offline-request-window-timeout" json:"offlineRequestWindowTimeout,omitempty" property:"offlineRequestWindowTimeout"`
	// true -> new request will be rejected.
	/// 新请求被拒绝
	RejectRequest atomic.Bool
	// active invocation
	// 活跃的调用
	ConsumerActiveCount atomic.Int32
	ProviderActiveCount atomic.Int32

	// provider last received request timestamp
	// provider 最后接受请求时间
	ProviderLastReceivedRequestTime atomic.Time

	// Closing 标记：区分"拒绝新请求"和"正在关闭"
	Closing atomic.Bool

	// Closing Invoker 过期时间
	ClosingInvokerExpireTime time.Duration `default:"30s" yaml:"closing-invoker-expire-time" json:"closingInvokerExpireTime,omitempty" property:"closingInvokerExpireTime"`
}

func DefaultShutdownConfig() *ShutdownConfig {
	cfg := &ShutdownConfig{}
	defaults.MustSet(cfg)

	return cfg
}

// Clone a new ShutdownConfig
func (c *ShutdownConfig) Clone() *ShutdownConfig {
	if c == nil {
		return nil
	}

	//InternalSignal 是一个指针类型 *bool，而不是值类型。
	//在 Go 语言中，直接赋值指针会导致两个变量指向同一个内存地址。
	var newInternalSignal *bool
	if c.InternalSignal != nil {
		newInternalSignal = new(bool) //new(bool) 初始化一个新的 bool 指针
		*newInternalSignal = *c.InternalSignal
	}

	newShutdownConfig := &ShutdownConfig{
		Timeout:                     c.Timeout,
		StepTimeout:                 c.StepTimeout,
		ConsumerUpdateWaitTime:      c.ConsumerUpdateWaitTime,
		RejectRequestHandler:        c.RejectRequestHandler,
		InternalSignal:              newInternalSignal,
		OfflineRequestWindowTimeout: c.OfflineRequestWindowTimeout,
		ClosingInvokerExpireTime:    c.ClosingInvokerExpireTime,
	}

	//原子类型字段（如 RejectRequest、ConsumerActiveCount 等）：使用 Store 方法拷贝值，确保线程安全
	newShutdownConfig.RejectRequest.Store(c.RejectRequest.Load())
	newShutdownConfig.ConsumerActiveCount.Store(c.ConsumerActiveCount.Load())
	newShutdownConfig.ProviderActiveCount.Store(c.ProviderActiveCount.Load())
	newShutdownConfig.ProviderLastReceivedRequestTime.Store(c.ProviderLastReceivedRequestTime.Load())
	newShutdownConfig.Closing.Store(c.Closing.Load())

	return newShutdownConfig
}
