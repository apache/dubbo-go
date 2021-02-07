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

package conncheck

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

// nolint
type MockInvoker struct {
	url *common.URL
}

// nolint
func NewMockInvoker(url *common.URL) *MockInvoker {
	return &MockInvoker{
		url: url,
	}
}

// nolint
func (bi *MockInvoker) GetUrl() *common.URL {
	return bi.url
}

// nolint
func (bi *MockInvoker) IsAvailable() bool {
	return true
}

// nolint
func (bi *MockInvoker) IsDestroyed() bool {
	return true
}

// nolint
func (bi *MockInvoker) Invoke(_ context.Context, _ protocol.Invocation) protocol.Result {
	return nil
}

// nolint
func (bi *MockInvoker) Destroy() {
}

// nolint
func TestHealthCheckRouteFactory(t *testing.T) {
	factory := newConnCheckRouteFactory()
	assert.NotNil(t, factory)
}
