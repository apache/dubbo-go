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

package grpc

import (
	"context"
	"reflect"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/protocol"

	hessian2 "github.com/apache/dubbo-go-hessian2"

	"github.com/pkg/errors"

	"google.golang.org/grpc/connectivity"
)

var errNoReply = errors.New("request need @response")

// nolint
type GrpcInvoker struct {
	protocol.BaseInvoker
	quitOnce    sync.Once
	clientGuard *sync.RWMutex
	client      *Client
}

// NewGrpcInvoker returns a Grpc invoker instance
func NewGrpcInvoker(url *common.URL, client *Client) *GrpcInvoker {
	return &GrpcInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		clientGuard: &sync.RWMutex{},
		client:      client,
	}
}

func (gi *GrpcInvoker) setClient(client *Client) {
	gi.clientGuard.Lock()
	defer gi.clientGuard.Unlock()

	gi.client = client
}

func (gi *GrpcInvoker) getClient() *Client {
	gi.clientGuard.RLock()
	defer gi.clientGuard.RUnlock()

	return gi.client
}

// Invoke is used to call service method by invocation
func (gi *GrpcInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	var result protocol.RPCResult

	if !gi.BaseInvoker.IsAvailable() {
		// Generally, the case will not happen, because the invoker has been removed
		// from the invoker list before destroy,so no new request will enter the destroyed invoker
		logger.Warnf("this grpcInvoker is destroyed")
		result.Err = protocol.ErrDestroyedInvoker
		return &result
	}

	gi.clientGuard.RLock()
	defer gi.clientGuard.RUnlock()

	if gi.client == nil {
		result.Err = protocol.ErrClientClosed
		return &result
	}

	if !gi.BaseInvoker.IsAvailable() {
		// Generally, the case will not happen, because the invoker has been removed
		// from the invoker list before destroy,so no new request will enter the destroyed invoker
		logger.Warnf("this grpcInvoker is destroying")
		result.Err = protocol.ErrDestroyedInvoker
		return &result
	}

	if invocation.Reply() == nil {
		result.Err = errNoReply
	}

	var in []reflect.Value
	in = append(in, reflect.ValueOf(ctx))
	in = append(in, invocation.ParameterValues()...)

	methodName := invocation.MethodName()
	method := gi.client.invoker.MethodByName(methodName)
	res := method.Call(in)

	result.Rest = res[0]
	// check err
	if !res[1].IsNil() {
		result.Err = res[1].Interface().(error)
	} else {
		_ = hessian2.ReflectResponse(res[0], invocation.Reply())
	}

	return &result
}

// IsAvailable get available status
func (gi *GrpcInvoker) IsAvailable() bool {
	client := gi.getClient()
	if client != nil {
		return gi.BaseInvoker.IsAvailable() && client.GetState() != connectivity.Shutdown
	}

	return false
}

// IsDestroyed get destroyed status
func (gi *GrpcInvoker) IsDestroyed() bool {
	client := gi.getClient()
	if client != nil {
		return gi.BaseInvoker.IsDestroyed() && client.GetState() == connectivity.Shutdown
	}

	return false
}

// Destroy will destroy gRPC's invoker and client, so it is only called once
func (gi *GrpcInvoker) Destroy() {
	gi.quitOnce.Do(func() {
		gi.BaseInvoker.Destroy()
		client := gi.getClient()
		if client != nil {
			gi.setClient(nil)
			client.Close()
		}
	})
}
