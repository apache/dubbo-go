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

package graceful_shutdown

import (
	"context"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

var (
	// psfOnce 确保 providerGracefulShutdownFilter 只被初始化一次
	psfOnce sync.Once
	// psf 提供方优雅关闭过滤器
	psf *providerGracefulShutdownFilter
)

func init() {
	// `init()` is performed before config.Load(), so shutdownConfig will be retrieved after config was loaded.
	extension.SetFilter(constant.GracefulShutdownProviderFilterKey, func() filter.Filter {
		return newProviderGracefulShutdownFilter() // 就是一个单例
	})
}

type providerGracefulShutdownFilter struct {
	// shutdownConfig 关闭配置
	shutdownConfig *global.ShutdownConfig
}

func newProviderGracefulShutdownFilter() filter.Filter {
	if psf == nil {
		psfOnce.Do(func() {
			psf = &providerGracefulShutdownFilter{}
		})
	}
	return psf
}

// Invoke adds the requests count and blocks the new requests if application is closing
// 会添加请求计数 & 如果应用关闭，并阻止新请求
func (f *providerGracefulShutdownFilter) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	// 如果拒绝新请求，则返回拒绝执行结果
	if f.rejectNewRequest() {
		logger.Info("The application is closing, new request will be rejected.")
		// 使用配置的拒绝处理器处理请求
		handler := constant.DefaultKey
		// 如果配置了拒绝请求处理器，则使用配置的拒绝处理器处理请求
		if f.shutdownConfig != nil && len(f.shutdownConfig.RejectRequestHandler) > 0 {
			handler = f.shutdownConfig.RejectRequestHandler
		}
		// 如果获取拒绝执行处理器失败，则返回默认拒绝执行结果
		rejectedExecutionHandler, err := extension.GetRejectedExecutionHandler(handler)
		if err != nil {
			logger.Warn(err)
		} else {
			// 调用拒绝执行处理器处理请求, 什么也不做, 只是记录日志
			return rejectedExecutionHandler.RejectedExecution(invoker.GetURL(), invocation)
		}
	}
	// 增加活跃请求计数
	f.shutdownConfig.ProviderActiveCount.Inc()
	// 更新最后接收请求的时间
	f.shutdownConfig.ProviderLastReceivedRequestTime.Store(time.Now())
	// 继续执行原始调用
	return invoker.Invoke(ctx, invocation)
}

// OnResponse reduces the number of active processes then return the process result
func (f *providerGracefulShutdownFilter) OnResponse(ctx context.Context, result result.Result, invoker base.Invoker, invocation base.Invocation) result.Result {
	// 处理响应，减少活跃请求计数
	f.shutdownConfig.ProviderActiveCount.Dec()

	// 在 Closing 状态下，响应携带 Closing 标记
	if f.isClosing() {
		result.AddAttachment(constant.GracefulShutdownClosingKey, "true")
	}

	return result
}

func (f *providerGracefulShutdownFilter) Set(name string, conf any) {
	switch name {
	// 设置过滤器配置
	case constant.GracefulShutdownFilterShutdownConfig:
		switch ct := conf.(type) {
		case *global.ShutdownConfig:
			f.shutdownConfig = ct
		// only for compatibility with old config, able to directly remove after config is deleted
		// 兼容旧配置，能够直接删除后
		case *config.ShutdownConfig:
			f.shutdownConfig = compatGlobalShutdownConfig(ct)
		default:
			logger.Warnf("the type of config for {%s} should be *global.ShutdownConfig", constant.GracefulShutdownFilterShutdownConfig)
		}
		return
	default:
		// do nothing
	}
}

func (f *providerGracefulShutdownFilter) rejectNewRequest() bool {
	if f.shutdownConfig == nil {
		return false
	}
	return f.shutdownConfig.RejectRequest.Load()
}

func (f *providerGracefulShutdownFilter) isClosing() bool {
	if f.shutdownConfig == nil {
		return false
	}
	return f.shutdownConfig.Closing.Load()
}
