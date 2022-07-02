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

package cluster

import (
	"context"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var Count int

type Rest struct {
	Tried   int
	Success bool
}

type mockCluster struct{}

// NewMockCluster returns a mock cluster instance.
//
// Mock cluster is usually used for service degradation, such as an authentication service.
// When the service provider is completely hung up, the client does not throw an exception,
// return an authorization failure through the Mock data instead.
func NewMockCluster() Cluster {
	return &mockCluster{}
}

func (cluster *mockCluster) Join(directory directory.Directory) protocol.Invoker {
	return BuildInterceptorChain(protocol.NewBaseInvoker(directory.GetURL()))
}

type MockInvoker struct {
	url       *common.URL
	available bool
	destroyed bool

	successCount int
}

func NewMockInvoker(url *common.URL, successCount int) *MockInvoker {
	return &MockInvoker{
		url:          url,
		available:    true,
		destroyed:    false,
		successCount: successCount,
	}
}

func (bi *MockInvoker) GetURL() *common.URL {
	return bi.url
}

func (bi *MockInvoker) IsAvailable() bool {
	return bi.available
}

func (bi *MockInvoker) IsDestroyed() bool {
	return bi.destroyed
}

func (bi *MockInvoker) Invoke(c context.Context, invocation protocol.Invocation) protocol.Result {
	Count++
	var (
		success bool
		err     error
	)
	if Count >= bi.successCount {
		success = true
	} else {
		err = perrors.New("error")
	}
	result := &protocol.RPCResult{Err: err, Rest: Rest{Tried: Count, Success: success}}

	return result
}

func (bi *MockInvoker) Destroy() {
	logger.Infof("Destroy invoker: %v", bi.GetURL().String())
	bi.destroyed = true
	bi.available = false
}
