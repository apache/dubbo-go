/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grpc

import (
	"sync"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

type GrpcInvoker struct {
	protocol.BaseInvoker
	client   *Client
	quitOnce sync.Once
}

func NewGrpcInvoker(url common.URL, client *Client) *GrpcInvoker {
	return &GrpcInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		client:      client,
	}
}

func (gi *GrpcInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	var (
		result protocol.RPCResult
	)

	return &result
}

func (gi *GrpcInvoker) Destroy() {
	gi.quitOnce.Do(func() {
		gi.BaseInvoker.Destroy()

		if gi.client != nil {
			gi.client.Close()
		}
	})
}
