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
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	grpcgo "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health "google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
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

	respA, errA := server.healthServer.Check(context.TODO(), &grpc_health_v1.HealthCheckRequest{Service: "svc-a"})
	respB, errB := server.healthServer.Check(context.TODO(), &grpc_health_v1.HealthCheckRequest{Service: "svc-b"})
	assert.NoError(t, errA)
	assert.NoError(t, errB)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, respA.GetStatus())
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, respB.GetStatus())
}

func TestGrpcHealthWatchEmitsClosingEvent(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	serviceKey := common.ServiceKey(constant.HealthCheckServiceInterface, "group", "1.0.0")
	healthServer := grpc_health.NewServer()
	healthServer.SetServingStatus(serviceKey, grpc_health_v1.HealthCheckResponse_SERVING)

	server := grpcgo.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)
	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Stop()

	url, err := common.NewURL(fmt.Sprintf(
		"grpc://%s/%s?interface=%s&group=group&version=1.0.0",
		listener.Addr().String(),
		constant.HealthCheckServiceInterface,
		constant.HealthCheckServiceInterface,
	))
	require.NoError(t, err)

	conn, err := grpcgo.Dial(
		listener.Addr().String(),
		grpcgo.WithTransportCredentials(insecure.NewCredentials()),
		grpcgo.WithBlock(),
		grpcgo.WithTimeout(3*time.Second),
	)
	require.NoError(t, err)
	defer conn.Close()

	invoker := &GrpcInvoker{
		BaseInvoker: *base.NewBaseInvoker(url),
		clientGuard: &sync.RWMutex{},
		client:      &Client{ClientConn: conn},
	}
	handler := &testClosingEventHandler{}
	invoker.startHealthWatch(handler)
	defer invoker.Destroy()

	healthServer.SetServingStatus(serviceKey, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	require.Eventually(t, func() bool {
		return len(handler.events) == 1
	}, 3*time.Second, 20*time.Millisecond)

	assert.Equal(t, "grpc-health-watch", handler.events[0].Source)
	assert.Equal(t, url.GetCacheInvokerMapKey(), handler.events[0].InstanceKey)
	assert.Equal(t, serviceKey, handler.events[0].ServiceKey)
}
