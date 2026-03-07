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
	"strconv"
	"strings"
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

// Invoke adds the requests count and checks if invoker is closing
func (f *consumerGracefulShutdownFilter) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	// check if invoker is closing
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

	// check closing flag in response
	if f.isClosingResponse(result) {
		f.markClosingInvoker(invoker)
	}

	// handle request error
	if result.Error() != nil {
		f.handleRequestError(invoker, result.Error())
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

// isClosingInvoker checks if invoker is in closing list
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

// isClosingResponse checks if response contains closing flag
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

// markClosingInvoker marks invoker as closing and sets available=false
func (f *consumerGracefulShutdownFilter) markClosingInvoker(invoker base.Invoker) {
	key := invoker.GetURL().String()
	expireTime := time.Now().Add(f.getClosingInvokerExpireTime())
	f.closingInvokers.Store(key, expireTime)

	logger.Infof("Graceful shutdown: marked invoker as closing: %s, will expire at %v, IsAvailable=%v",
		key, expireTime, invoker.IsAvailable())

	if bi, ok := invoker.(*base.BaseInvoker); ok {
		bi.SetAvailable(false)
		logger.Infof("Graceful shutdown: set invoker unavailable: %s, IsAvailable now=%v",
			key, invoker.IsAvailable())
	}
}

func (f *consumerGracefulShutdownFilter) getClosingInvokerExpireTime() time.Duration {
	if f.shutdownConfig != nil && f.shutdownConfig.ClosingInvokerExpireTime != "" {
		if duration, err := time.ParseDuration(f.shutdownConfig.ClosingInvokerExpireTime); err == nil && duration > 0 {
			return duration
		}
	}
	// default 30s, also try parsing numeric string as milliseconds
	if f.shutdownConfig != nil && f.shutdownConfig.ClosingInvokerExpireTime != "" {
		if ms, err := strconv.ParseInt(f.shutdownConfig.ClosingInvokerExpireTime, 10, 64); err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return 30 * time.Second
}

// handleRequestError handles request errors and marks invoker as unavailable for connection errors
func (f *consumerGracefulShutdownFilter) handleRequestError(invoker base.Invoker, err error) {
	if err == nil {
		return
	}

	// check for connection-related errors
	errMsg := err.Error()
	isConnectionError := strings.Contains(errMsg, "client has closed") ||
		strings.Contains(errMsg, "connection") ||
		strings.Contains(errMsg, "EOF") ||
		strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "gRPC") && strings.Contains(errMsg, "closing") ||
		strings.Contains(errMsg, "http2") && strings.Contains(errMsg, "close")

	if isConnectionError {
		key := invoker.GetURL().String()
		expireTime := time.Now().Add(f.getClosingInvokerExpireTime())
		f.closingInvokers.Store(key, expireTime)

		logger.Infof("Graceful shutdown: connection error detected for invoker: %s, marking as closing, will expire at %v, IsAvailable=%v",
			key, expireTime, invoker.IsAvailable())

		if bi, ok := invoker.(*base.BaseInvoker); ok {
			bi.SetAvailable(false)
			logger.Infof("Graceful shutdown: set invoker unavailable due to connection error: %s, IsAvailable now=%v",
				key, invoker.IsAvailable())
		}
	}
}
