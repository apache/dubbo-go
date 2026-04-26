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

	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	gracefulshutdown "dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

type testTripleClosingEventHandler struct {
	lock   sync.Mutex
	events []gracefulshutdown.ClosingEvent
}

func (h *testTripleClosingEventHandler) HandleClosingEvent(event gracefulshutdown.ClosingEvent) bool {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.events = append(h.events, event)
	return true
}

// Events returns a snapshot because health-watch callbacks run in a goroutine.
func (h *testTripleClosingEventHandler) Events() []gracefulshutdown.ClosingEvent {
	h.lock.Lock()
	defer h.lock.Unlock()

	events := make([]gracefulshutdown.ClosingEvent, len(h.events))
	copy(events, h.events)
	return events
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
	events := handler.Events()
	if assert.Len(t, events, 1) {
		assert.Equal(t, "triple-health-watch", events[0].Source)
		assert.Equal(t, url.GetCacheInvokerMapKey(), events[0].InstanceKey)
		assert.Equal(t, url.ServiceKey(), events[0].ServiceKey)
	}
}

func TestTripleHealthWatchEmitsClosingEvent(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	_ = listener.Close()

	serviceKey := common.ServiceKey(constant.HealthCheckServiceInterface, "group", "1.0.0")
	notServing := make(chan struct{})

	server := tri.NewServer(addr, nil)
	err = server.RegisterServerStreamHandler(
		"/grpc.health.v1.Health/Watch",
		func() any { return new(grpc_health_v1.HealthCheckRequest) },
		func(ctx context.Context, req *tri.Request, stream *tri.ServerStream) error {
			request, ok := req.Msg.(*grpc_health_v1.HealthCheckRequest)
			if !ok {
				return fmt.Errorf("unexpected request type %T", req.Msg)
			}
			if request.GetService() != serviceKey {
				return fmt.Errorf("unexpected service %s", request.GetService())
			}
			if sendErr := stream.Send(&grpc_health_v1.HealthCheckResponse{
				Status: grpc_health_v1.HealthCheckResponse_SERVING,
			}); sendErr != nil {
				return sendErr
			}
			select {
			case <-notServing:
			case <-ctx.Done():
				return ctx.Err()
			}
			return stream.Send(&grpc_health_v1.HealthCheckResponse{
				Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
			})
		},
	)
	require.NoError(t, err)

	go func() {
		_ = server.Run(constant.CallHTTP2, nil)
	}()
	defer func() {
		_ = server.Stop()
	}()

	require.Eventually(t, func() bool {
		conn, dialErr := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if dialErr != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 3*time.Second, 20*time.Millisecond)

	url, err := common.NewURL(fmt.Sprintf(
		"tri://%s/%s?interface=%s&group=group&version=1.0.0&timeout=1s",
		addr,
		constant.HealthCheckServiceInterface,
		constant.HealthCheckServiceInterface,
	))
	require.NoError(t, err)

	cm, err := newClientManager(url)
	require.NoError(t, err)
	invoker := &TripleInvoker{
		BaseInvoker:   *base.NewBaseInvoker(url),
		clientGuard:   &sync.RWMutex{},
		clientManager: cm,
	}
	defer invoker.Destroy()

	handler := &testTripleClosingEventHandler{}
	invoker.startHealthWatch(handler)

	close(notServing)

	require.Eventually(t, func() bool {
		return len(handler.Events()) == 1
	}, 3*time.Second, 20*time.Millisecond)

	events := handler.Events()
	assert.Equal(t, "triple-health-watch", events[0].Source)
	assert.Equal(t, url.GetCacheInvokerMapKey(), events[0].InstanceKey)
	assert.Equal(t, serviceKey, events[0].ServiceKey)
}
