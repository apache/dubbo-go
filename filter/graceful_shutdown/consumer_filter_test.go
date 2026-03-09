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
	"net/url"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

type testEmbeddedInvoker struct {
	base.BaseInvoker
}

func newTestEmbeddedInvoker(rawURL *common.URL) *testEmbeddedInvoker {
	return &testEmbeddedInvoker{
		BaseInvoker: *base.NewBaseInvoker(rawURL),
	}
}

func (i *testEmbeddedInvoker) Invoke(ctx context.Context, invocation base.Invocation) result.Result {
	return &result.RPCResult{}
}

func TestConsumerFilterInvokeWithGlobalPackage(t *testing.T) {
	var (
		baseUrl        = common.NewURLWithOptions(common.WithParams(url.Values{}))
		rpcInvocation  = invocation.NewRPCInvocation("GetUser", []any{"OK"}, make(map[string]any))
		opt            = graceful_shutdown.NewOptions()
		filterValue, _ = extension.GetFilter(constant.GracefulShutdownConsumerFilterKey)
	)

	opt.Shutdown.RejectRequestHandler = "test"

	filter := filterValue.(*consumerGracefulShutdownFilter)
	filter.Set(constant.GracefulShutdownFilterShutdownConfig, opt.Shutdown)
	assert.Equal(t, filter.shutdownConfig, opt.Shutdown)

	result := filter.Invoke(context.Background(), base.NewBaseInvoker(baseUrl), rpcInvocation)
	assert.NotNil(t, result)
	assert.NoError(t, result.Error())
}

func TestIsClosingError(t *testing.T) {
	assert.True(t, isClosingError(base.ErrClientClosed))
	assert.True(t, isClosingError(errors.New("rpc error: code = Unavailable desc = transport is closing")))
	assert.False(t, isClosingError(errors.New("EOF")))
	assert.False(t, isClosingError(errors.New("read tcp: connection reset by peer")))
}

func TestMarkClosingInvokerSetsEmbeddedInvokerUnavailable(t *testing.T) {
	filter := &consumerGracefulShutdownFilter{
		shutdownConfig: graceful_shutdown.NewOptions().Shutdown,
	}
	invoker := newTestEmbeddedInvoker(common.NewURLWithOptions(common.WithParams(url.Values{})))

	assert.True(t, invoker.IsAvailable())

	filter.markClosingInvoker(invoker)

	assert.False(t, invoker.IsAvailable())
	expireTime, ok := filter.closingInvokers.Load(invoker.GetURL().String())
	assert.True(t, ok)
	assert.True(t, expireTime.(time.Time).After(time.Now()))
}

func TestConsumerFilterDoesNotDecrementWithoutIncrement(t *testing.T) {
	filter := &consumerGracefulShutdownFilter{
		shutdownConfig: graceful_shutdown.NewOptions().Shutdown,
	}
	invoker := newTestEmbeddedInvoker(common.NewURLWithOptions(common.WithParams(url.Values{})))
	rpcInvocation := invocation.NewRPCInvocation("GetUser", []any{"OK"}, make(map[string]any))

	filter.markClosingInvoker(invoker)

	res := filter.Invoke(context.Background(), invoker, rpcInvocation)
	assert.Error(t, res.Error())
	assert.Equal(t, "provider is closing", res.Error().Error())

	filter.OnResponse(context.Background(), res, invoker, rpcInvocation)
	assert.Equal(t, int32(0), filter.shutdownConfig.ConsumerActiveCount.Load())
}

func TestClosingInvokerExpiryRestoresAvailability(t *testing.T) {
	opt := graceful_shutdown.NewOptions()
	opt.Shutdown.ClosingInvokerExpireTime = "20ms"

	filter := &consumerGracefulShutdownFilter{
		shutdownConfig: opt.Shutdown,
	}
	invoker := newTestEmbeddedInvoker(common.NewURLWithOptions(common.WithParams(url.Values{})))

	filter.markClosingInvoker(invoker)
	assert.False(t, invoker.IsAvailable())
	assert.True(t, filter.isClosingInvoker(invoker))

	time.Sleep(40 * time.Millisecond)
	assert.False(t, filter.isClosingInvoker(invoker))
	assert.True(t, invoker.IsAvailable())
}

func TestHandleRequestErrorDoesNotMarkNonClosingErrors(t *testing.T) {
	filter := &consumerGracefulShutdownFilter{
		shutdownConfig: graceful_shutdown.NewOptions().Shutdown,
	}
	invoker := newTestEmbeddedInvoker(common.NewURLWithOptions(common.WithParams(url.Values{})))

	filter.handleRequestError(invoker, errors.New("EOF"))
	_, ok := filter.closingInvokers.Load(invoker.GetURL().String())
	assert.False(t, ok)
	assert.True(t, invoker.IsAvailable())

	filter.handleRequestError(invoker, errors.New("read tcp: connection reset by peer"))
	_, ok = filter.closingInvokers.Load(invoker.GetURL().String())
	assert.False(t, ok)
	assert.True(t, invoker.IsAvailable())
}
