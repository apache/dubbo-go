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
	shutdownConfig *config.ShutdownConfig
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
func (f *consumerGracefulShutdownFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	f.shutdownConfig.ConsumerActiveCount.Inc()
	return invoker.Invoke(ctx, invocation)
}

// OnResponse reduces the number of active processes then return the process result
func (f *consumerGracefulShutdownFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	f.shutdownConfig.ConsumerActiveCount.Dec()
	return result
}

func (f *consumerGracefulShutdownFilter) Set(name string, conf interface{}) {
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
