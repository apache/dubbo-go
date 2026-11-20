/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package proxy_factory

import (
	"context"
	"errors"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

var errBizFixture = errors.New("insufficient balance")

type ClassificationService struct{}

func (s *ClassificationService) Fail(_ context.Context, _ string) (string, error) {
	return "", errBizFixture
}

func (s *ClassificationService) Expire(_ context.Context, _ string) (string, error) {
	return "", context.DeadlineExceeded
}

func (s *ClassificationService) Crash(_ context.Context, _ string) (string, error) {
	panic("fixture panic")
}

// invokeObserved runs the invoker like the provider metrics filter does and
// returns the result together with the kind recorded on the observation.
func invokeObserved(t *testing.T, invoker base.Invoker, method string) (rpcerror.Kind, error) {
	t.Helper()
	ctx, o := rpcerror.WithObservation(context.Background())
	got := make(chan rpcerror.Completion, 1)
	o.OnComplete(func(c rpcerror.Completion) { got <- c })
	res := invoker.Invoke(ctx, invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName(method),
		invocation.WithArguments([]any{"x"}),
	))
	o.FinishInvoke(res.Error(), true)
	select {
	case c := <-got:
		return c.Kind, res.Error()
	case <-time.After(time.Second):
		t.Fatal("observation did not complete")
		return rpcerror.KindUnknown, nil
	}
}

func TestProxyInvokerClassifiesProviderFailures(t *testing.T) {
	const (
		protocol      = "classification-protocol"
		interfaceName = "ClassificationService"
	)
	registerService(t, protocol, interfaceName, &ClassificationService{})
	invoker := &ProxyInvoker{BaseInvoker: *base.NewBaseInvoker(newURL(protocol, interfaceName))}

	t.Run("application error is business and stays unwrapped", func(t *testing.T) {
		kind, err := invokeObserved(t, invoker, "Fail")
		require.Same(t, errBizFixture, err, "protocols re-encode the exact provider error")
		assert.Equal(t, rpcerror.KindBusiness, kind)
	})

	t.Run("expired context deadline stays a timeout", func(t *testing.T) {
		kind, err := invokeObserved(t, invoker, "Expire")
		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Equal(t, rpcerror.KindUnknown, kind, "business must not override the deadline")
		assert.Equal(t, rpcerror.KindTimeout, rpcerror.KindOf(rpcerror.WithKind(err, kind)))
	})

	t.Run("panic is not counted as business", func(t *testing.T) {
		kind, err := invokeObserved(t, invoker, "Crash")
		require.Error(t, err)
		assert.Equal(t, rpcerror.KindUnknown, kind)
		assert.Equal(t, rpcerror.KindUnknown, rpcerror.KindOf(err))
	})

	t.Run("missing method is service unavailable", func(t *testing.T) {
		kind, err := invokeObserved(t, invoker, "Missing")
		assert.Equal(t, rpcerror.KindServiceUnavailable, rpcerror.KindOf(err))
		assert.Equal(t, rpcerror.KindUnknown, kind)
		assert.Contains(t, err.Error(), "cannot find method [Missing]")
	})

	t.Run("missing service is service unavailable", func(t *testing.T) {
		absent := &ProxyInvoker{BaseInvoker: *base.NewBaseInvoker(newURL(protocol, "UnknownService"))}
		_, err := invokeObserved(t, absent, "Fail")
		assert.Equal(t, rpcerror.KindServiceUnavailable, rpcerror.KindOf(err))
		assert.Contains(t, err.Error(), "cannot find service [UnknownService]")
	})

	t.Run("info proxy missing method is service unavailable", func(t *testing.T) {
		info := &infoProxyInvoker{BaseInvoker: *base.NewBaseInvoker(newURL(protocol, interfaceName))}
		res := info.Invoke(context.Background(), invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("Missing")))
		assert.Equal(t, rpcerror.KindServiceUnavailable, rpcerror.KindOf(res.Error()))
	})
}
