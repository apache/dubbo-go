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
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
)

//go:generate mockgen -source invoker.go -destination mock/mock_invoker.go  -self_package github.com/apache/dubbo-go/protocol/mock --package mock  Invoker
// Extension - Invoker
type Invoker interface {
	common.Node
	Invoke(Invocation) Result
}

/////////////////////////////
// base invoker
/////////////////////////////

type BaseInvoker struct {
	url       common.URL
	available bool
	destroyed bool
}

func NewBaseInvoker(url common.URL) *BaseInvoker {
	return &BaseInvoker{
		url:       url,
		available: true,
		destroyed: false,
	}
}

func (bi *BaseInvoker) GetUrl() common.URL {
	return bi.url
}

func (bi *BaseInvoker) IsAvailable() bool {
	return bi.available
}

func (bi *BaseInvoker) IsDestroyed() bool {
	return bi.destroyed
}

func (bi *BaseInvoker) Invoke(invocation Invocation) Result {
	return &RPCResult{}
}

func (bi *BaseInvoker) Destroy() {
	logger.Infof("Destroy invoker: %s", bi.GetUrl().String())
	bi.destroyed = true
	bi.available = false
}
