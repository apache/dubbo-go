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
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	psfOnce sync.Once
	psf     *providerGracefulShutdownFilter
)

func init() {
	// `init()` is performed before config.Load(), so shutdownConfig will be retrieved after config was loaded.
	extension.SetFilter(constant.GracefulShutdownProviderFilterKey, func() filter.Filter {
		return newProviderGracefulShutdownFilter()
	})
}

type providerGracefulShutdownFilter struct {
	shutdownConfig *config.ShutdownConfig
}

func newProviderGracefulShutdownFilter() filter.Filter {
	if psf == nil {
		psfOnce.Do(func() {
			psf = &providerGracefulShutdownFilter{}
		})
	}
	return psf
}

// Invoke adds the requests count and block the new requests if application is closing
func (f *providerGracefulShutdownFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if f.rejectNewRequest() {
		logger.Info("The application is closing, new request will be rejected.")
		handler := constant.DefaultKey
		if f.shutdownConfig != nil && len(f.shutdownConfig.RejectRequestHandler) > 0 {
			handler = f.shutdownConfig.RejectRequestHandler
		}
		rejectedExecutionHandler, err := extension.GetRejectedExecutionHandler(handler)
		if err != nil {
			logger.Warn(err)
		} else {
			return rejectedExecutionHandler.RejectedExecution(invoker.GetURL(), invocation)
		}
	}
	f.shutdownConfig.ProviderActiveCount.Inc()
	return invoker.Invoke(ctx, invocation)
}

// OnResponse reduces the number of active processes then return the process result
func (f *providerGracefulShutdownFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	f.shutdownConfig.ProviderActiveCount.Dec()
	return result
}

func (f *providerGracefulShutdownFilter) Set(name string, conf interface{}) {
	switch name {
	case constant.GracefulShutdownFilterShutdownConfig:
		if shutdownConfig, ok := conf.(*config.ShutdownConfig); ok {
			f.shutdownConfig = shutdownConfig
			return
		}
		logger.Warnf("the type of config for {%s} should be *config.ShutdownConfig", constant.GracefulShutdownFilterShutdownConfig)
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
