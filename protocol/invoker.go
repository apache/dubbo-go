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
	"fmt"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"

	uatomic "go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

var (
	ErrClientClosed     = perrors.New("remoting client has closed")
	ErrNoReply          = perrors.New("request need @response")
	ErrDestroyedInvoker = perrors.New("request Destroyed invoker")
)

// Invoker the service invocation interface for the consumer
//go:generate mockgen -source invoker.go -destination mock/mock_invoker.go -self_package dubbo.apache.org/dubbo-go/v3/protocol/mock --package mock Invoker
// Extension - Invoker
type Invoker interface {
	common.Node
	// Invoke the invocation and return result.
	Invoke(context.Context, Invocation) Result
}

// BaseInvoker provides default invoker implements Invoker
type BaseInvoker struct {
	url       *common.URL
	available uatomic.Bool
	destroyed uatomic.Bool
}

// NewBaseInvoker creates a new BaseInvoker
func NewBaseInvoker(url *common.URL) *BaseInvoker {
	ivk := &BaseInvoker{
		url: url,
	}
	ivk.available.Store(true)
	ivk.destroyed.Store(false)

	return ivk
}

// GetURL gets base invoker URL
func (bi *BaseInvoker) GetURL() *common.URL {
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

// Invoke provides default invoker implement
func (bi *BaseInvoker) Invoke(context context.Context, invocation Invocation) Result {
	return &RPCResult{}
}

// Destroy changes available and destroyed flag
func (bi *BaseInvoker) Destroy() {
	logger.Infof("Destroy invoker: %s", bi.GetURL())
	bi.destroyed.Store(true)
	bi.available.Store(false)
}

func (bi *BaseInvoker) String() string {
	if bi.url != nil {
		return fmt.Sprintf("invoker{protocol: %s, host: %s:%s, path: %s}",
			bi.url.Protocol, bi.url.Ip, bi.url.Port, bi.url.Path)
	}
	return fmt.Sprintf("%#v", bi)
}
