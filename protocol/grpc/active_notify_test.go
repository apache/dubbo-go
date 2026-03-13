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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	gracefulshutdown "dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

type testClosingEventHandler struct {
	events []gracefulshutdown.ClosingEvent
}

func (h *testClosingEventHandler) HandleClosingEvent(event gracefulshutdown.ClosingEvent) bool {
	h.events = append(h.events, event)
	return true
}

func TestGrpcInvokerHandleHealthStatusNotServing(t *testing.T) {
	url, err := common.NewURL(helloworldURL)
	if err != nil {
		t.Fatal(err)
	}
	invoker := &GrpcInvoker{BaseInvoker: *base.NewBaseInvoker(url)}
	handler := &testClosingEventHandler{}

	handled := invoker.handleHealthStatus(grpc_health_v1.HealthCheckResponse_NOT_SERVING, handler)

	assert.True(t, handled)
	if assert.Len(t, handler.events, 1) {
		assert.Equal(t, "grpc-health-watch", handler.events[0].Source)
		assert.Equal(t, url.GetCacheInvokerMapKey(), handler.events[0].InstanceKey)
		assert.Equal(t, url.ServiceKey(), handler.events[0].ServiceKey)
	}
}

func TestGrpcServerSetAllServicesNotServing(t *testing.T) {
	server := NewServer()
	server.SetServingStatus("svc-a", grpc_health_v1.HealthCheckResponse_SERVING)
	server.SetServingStatus("svc-b", grpc_health_v1.HealthCheckResponse_SERVING)

	server.SetAllServicesNotServing()

	respA, errA := server.healthServer.Check(nil, &grpc_health_v1.HealthCheckRequest{Service: "svc-a"})
	respB, errB := server.healthServer.Check(nil, &grpc_health_v1.HealthCheckRequest{Service: "svc-b"})
	assert.NoError(t, errA)
	assert.NoError(t, errB)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, respA.GetStatus())
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, respB.GetStatus())
}
