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

package gshutdown

import (
	"context"
	"sync/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

func init() {
	// `init()` is performed before config.Load(), so shutdownConfig will be retrieved after config was loaded.
	extension.SetFilter(constant.GracefulShutdownConsumerFilterKey, func() filter.Filter {
		return &Filter{}
	})
	extension.SetFilter(constant.GracefulShutdownProviderFilterKey, func() filter.Filter {
		return &Filter{}
	})
}

type Filter struct {
	activeCount    int32
	shutdownConfig *config.ShutdownConfig
}

// Invoke adds the requests count and block the new requests if application is closing
func (f *Filter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if f.rejectNewRequest() {
		logger.Info("The application is closing, new request will be rejected.")
		return f.getRejectHandler().RejectedExecution(invoker.GetURL(), invocation)
	}
	atomic.AddInt32(&f.activeCount, 1)
	return invoker.Invoke(ctx, invocation)
}

// OnResponse reduces the number of active processes then return the process result
func (f *Filter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	atomic.AddInt32(&f.activeCount, -1)
	// although this isn't thread safe, it won't be a problem if the f.rejectNewRequest() is true.
	if f.shutdownConfig != nil && f.shutdownConfig.RejectRequest && f.activeCount <= 0 {
		f.shutdownConfig.RequestsFinished = true
	}
	return result
}

func (f *Filter) Set(name string, conf interface{}) {
	switch name {
	case config.GracefulShutdownFilterShutdownConfig:
		if shutdownConfig, ok := conf.(*config.ShutdownConfig); !ok {
			f.shutdownConfig = shutdownConfig
			return
		}
		logger.Warnf("the type of config for {%s} should be *config.ShutdownConfig", config.GracefulShutdownFilterShutdownConfig)
	default:
		// do nothing
	}
}

func (f *Filter) rejectNewRequest() bool {
	if f.shutdownConfig == nil {
		return false
	}
	return f.shutdownConfig.RejectRequest
}

func (f *Filter) getRejectHandler() filter.RejectedExecutionHandler {
	handler := constant.DEFAULT_KEY
	if f.shutdownConfig != nil && len(f.shutdownConfig.RejectRequestHandler) > 0 {
		handler = f.shutdownConfig.RejectRequestHandler
	}
	return extension.GetRejectedExecutionHandler(handler)
}
