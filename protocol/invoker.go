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

package protocol

import (
	"context"
)

import (
	perrors "github.com/pkg/errors"
	uatomic "go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
)

var (
	// ErrClientClosed means client has clossed.
	ErrClientClosed = perrors.New("remoting client has closed")
	// ErrNoReply
	ErrNoReply = perrors.New("request need @response")
	// ErrDestroyedInvoker
	ErrDestroyedInvoker = perrors.New("request Destroyed invoker")
)

// Invoker the service invocation interface for the consumer
//go:generate mockgen -source invoker.go -destination mock/mock_invoker.go  -self_package github.com/apache/dubbo-go/protocol/mock --package mock  Invoker
// Extension - Invoker
type Invoker interface {
	common.Node
	// Invoke the invocation and return result.
	Invoke(context.Context, Invocation) Result
}

/////////////////////////////
// base invoker
/////////////////////////////

// BaseInvoker provides default invoker implement
type BaseInvoker struct {
	url       *common.URL
	available uatomic.Bool
	destroyed uatomic.Bool
	// Used to record the number of requests. -1 represent this invoker is destroyed
	ivkNum uatomic.Int64
}

// NewBaseInvoker creates a new BaseInvoker
func NewBaseInvoker(url *common.URL) *BaseInvoker {
	ivk := &BaseInvoker{
		url: url,
	}
	ivk.available.Store(true)
	ivk.destroyed.Store(false)
	ivk.ivkNum.Store(0)

	return ivk
}

// GetUrl gets base invoker URL
func (bi *BaseInvoker) GetUrl() *common.URL {
	return bi.url
}

// IsAvailable gets available flag
func (bi *BaseInvoker) IsAvailable() bool {
	return bi.available.Load()
}

// IsDestroyed gets destroyed flag
func (bi *BaseInvoker) IsDestroyed() bool {
	return bi.destroyed.Load()
}

// InvokeTimes atomically loads the wrapped value and return the invoke times.
func (bi *BaseInvoker) InvokeTimes() int64 {
	return bi.ivkNum.Load()
}

// AddInvokerTimes atomically adds to the wrapped int64 and returns the new value.
func (bi *BaseInvoker) AddInvokerTimes(num int64) int64 {
	return bi.ivkNum.Add(num)
}

// Invoke provides default invoker implement
func (bi *BaseInvoker) Invoke(context context.Context, invocation Invocation) Result {
	return &RPCResult{}
}

// Stop changes available flag
func (bi *BaseInvoker) Stop() {
	logger.Infof("Stop invoker: %s", bi.GetUrl())
	bi.available.Store(false)
}

// Destroy changes available and destroyed flag
func (bi *BaseInvoker) Destroy() {
	logger.Infof("Destroy invoker: %s", bi.GetUrl())
	bi.destroyed.Store(true)
	bi.available.Store(false)
}
