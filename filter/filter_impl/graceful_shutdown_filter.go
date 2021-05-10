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

package filter_impl

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
	consumerFiler := &gracefulShutdownFilter{
		shutdownConfig: config.GetConsumerConfig().ShutdownConfig,
	}
	providerFilter := &gracefulShutdownFilter{
		shutdownConfig: config.GetProviderConfig().ShutdownConfig,
	}

	extension.SetFilter(constant.CONSUMER_SHUTDOWN_FILTER, func() filter.Filter {
		return consumerFiler
	})

	extension.SetFilter(constant.PROVIDER_SHUTDOWN_FILTER, func() filter.Filter {
		return providerFilter
	})
}

type gracefulShutdownFilter struct {
	activeCount    int32
	shutdownConfig *config.ShutdownConfig
}

// Invoke adds the requests count and block the new requests if application is closing
func (gf *gracefulShutdownFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if gf.rejectNewRequest() {
		logger.Info("The application is closing, new request will be rejected.")
		return gf.getRejectHandler().RejectedExecution(invoker.GetURL(), invocation)
	}
	atomic.AddInt32(&gf.activeCount, 1)
	return invoker.Invoke(ctx, invocation)
}

// OnResponse reduces the number of active processes then return the process result
func (gf *gracefulShutdownFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	atomic.AddInt32(&gf.activeCount, -1)
	// although this isn't thread safe, it won't be a problem if the gf.rejectNewRequest() is true.
	if gf.shutdownConfig != nil && gf.activeCount <= 0 {
		gf.shutdownConfig.RequestsFinished = true
	}
	return result
}

func (gf *gracefulShutdownFilter) rejectNewRequest() bool {
	if gf.shutdownConfig == nil {
		return false
	}
	return gf.shutdownConfig.RejectRequest
}

func (gf *gracefulShutdownFilter) getRejectHandler() filter.RejectedExecutionHandler {
	handler := constant.DEFAULT_KEY
	if gf.shutdownConfig != nil && len(gf.shutdownConfig.RejectRequestHandler) > 0 {
		handler = gf.shutdownConfig.RejectRequestHandler
	}
	return extension.GetRejectedExecutionHandler(handler)
}
