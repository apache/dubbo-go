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
	"errors"
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
	csfOnce sync.Once
	csf     *consumerGracefulShutdownFilter
)

func init() {
	// `init()` is performed before config.Load(), so shutdownConfig will be retrieved after config was loaded.
	extension.SetFilter(constant.GracefulShutdownConsumerFilterKey, func() filter.Filter {
		return newConsumerGracefulShutdownFilter()
	})

}

type consumerGracefulShutdownFilter struct {
	shutdownConfig  *global.ShutdownConfig
	closingInvokers sync.Map // map[string]time.Time (url key -> expire time)
}

func newConsumerGracefulShutdownFilter() filter.Filter {
	if csf == nil {
		csfOnce.Do(func() {
			csf = &consumerGracefulShutdownFilter{}
		})
	}
	return csf
}

// Invoke adds the requests count and block the new requests if application is closing
func (f *consumerGracefulShutdownFilter) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	// 检查 Invoker 是否正在关闭
	if f.isClosingInvoker(invoker) {
		logger.Warnf("Graceful shutdown: skipping closing invoker: %s", invoker.GetURL().String())
		return &result.RPCResult{Err: errors.New("provider is closing")}
	}
	f.shutdownConfig.ConsumerActiveCount.Inc()
	return invoker.Invoke(ctx, invocation)
}

// OnResponse reduces the number of active processes then return the process result
func (f *consumerGracefulShutdownFilter) OnResponse(ctx context.Context, result result.Result, invoker base.Invoker, invocation base.Invocation) result.Result {
	f.shutdownConfig.ConsumerActiveCount.Dec()

	// 检测响应中的 Closing 标记
	if f.isClosingResponse(result) {
		f.markClosingInvoker(invoker)
	}

	return result
}

func (f *consumerGracefulShutdownFilter) Set(name string, conf any) {
	switch name {
	case constant.GracefulShutdownFilterShutdownConfig:
		switch ct := conf.(type) {
		case *global.ShutdownConfig:
			f.shutdownConfig = ct
		// only for compatibility with old config, able to directly remove after config is deleted
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

// isClosingInvoker 检查 Invoker 是否在 closing 列表中
func (f *consumerGracefulShutdownFilter) isClosingInvoker(invoker base.Invoker) bool {
	key := invoker.GetURL().String()
	if expireTime, ok := f.closingInvokers.Load(key); ok {
		if time.Now().Before(expireTime.(time.Time)) {
			return true
		}
		f.closingInvokers.Delete(key)
	}
	return false
}

// isClosingResponse 检查响应是否携带 Closing 标记
func (f *consumerGracefulShutdownFilter) isClosingResponse(result result.Result) bool {
	if result != nil && result.Attachments() != nil {
		if v, ok := result.Attachments()[constant.GracefulShutdownClosingKey]; ok {
			if v == "true" {
				return true
			}
		}
	}
	return false
}

// markClosingInvoker 标记 Invoker 为正在关闭
func (f *consumerGracefulShutdownFilter) markClosingInvoker(invoker base.Invoker) {
	key := invoker.GetURL().String()
	expireTime := time.Now().Add(f.getClosingInvokerExpireTime())
	f.closingInvokers.Store(key, expireTime)
	logger.Infof("Graceful shutdown: marked invoker as closing: %s, will expire at %v", key, expireTime)
}

func (f *consumerGracefulShutdownFilter) getClosingInvokerExpireTime() time.Duration {
	if f.shutdownConfig != nil && f.shutdownConfig.ClosingInvokerExpireTime > 0 {
		return f.shutdownConfig.ClosingInvokerExpireTime
	}
	return 30 * time.Second
}
