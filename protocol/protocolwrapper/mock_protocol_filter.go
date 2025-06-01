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

package protocolwrapper

import (
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

type mockProtocolFilter struct{}

// NewMockProtocolFilter creates a new mock protocol
func NewMockProtocolFilter() base.Protocol {
	return &mockProtocolFilter{}
}

// Export mock service for  remote invocation
func (pfw *mockProtocolFilter) Export(invoker base.Invoker) base.Exporter {
	return base.NewBaseExporter("key", invoker, &sync.Map{})
}

// Refer a mock remote service
func (pfw *mockProtocolFilter) Refer(url *common.URL) base.Invoker {
	return base.NewBaseInvoker(url)
}

// Destroy will do nothing
func (pfw *mockProtocolFilter) Destroy() {
}
