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

package triple

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	gracefulshutdown "dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type testTripleClosingEventHandler struct {
	events []gracefulshutdown.ClosingEvent
}

func (h *testTripleClosingEventHandler) HandleClosingEvent(event gracefulshutdown.ClosingEvent) bool {
	h.events = append(h.events, event)
	return true
}

func TestTripleInvokerHandleHealthStatusNotServing(t *testing.T) {
	url, err := common.NewURL("tri://127.0.0.1:20000/org.apache.dubbo-go.mockService?group=group&version=1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	invoker := &TripleInvoker{BaseInvoker: *base.NewBaseInvoker(url)}
	handler := &testTripleClosingEventHandler{}

	handled := invoker.handleHealthStatus(grpc_health_v1.HealthCheckResponse_NOT_SERVING, handler)

	assert.True(t, handled)
	if assert.Len(t, handler.events, 1) {
		assert.Equal(t, "triple-health-watch", handler.events[0].Source)
		assert.Equal(t, url.GetCacheInvokerMapKey(), handler.events[0].InstanceKey)
		assert.Equal(t, url.ServiceKey(), handler.events[0].ServiceKey)
	}
}
