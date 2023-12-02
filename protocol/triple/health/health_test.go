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

package health

import (
	"sync"
	"testing"
	"time"
)
import (
	"github.com/stretchr/testify/assert"

	// If there is a conflict between the healthCheck of Dubbo and the healthCheck of gRPC, an error will occur.
	_ "google.golang.org/grpc/health/grpc_health_v1"
)

import (
	healthpb "dubbo.apache.org/dubbo-go/v3/protocol/triple/health/triple_health"
)

const testService = "testService"

func TestSetServingStatus(t *testing.T) {
	s := NewServer()
	s.SetServingStatus(testService, healthpb.HealthCheckResponse_SERVING)

	status := s.statusMap[testService]
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, status, "status for %s is %v, want %v", testService, status, healthpb.HealthCheckResponse_SERVING)

	s.SetServingStatus(testService, healthpb.HealthCheckResponse_NOT_SERVING)
	status = s.statusMap[testService]
	assert.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, status, "status for %s is %v, want %v", testService, status, healthpb.HealthCheckResponse_NOT_SERVING)
}

func TestShutdown(t *testing.T) {
	s := NewServer()
	s.SetServingStatus(testService, healthpb.HealthCheckResponse_SERVING)
	var wg sync.WaitGroup
	wg.Add(2)
	// Run SetServingStatus and Shutdown in parallel.
	go func() {
		for i := 0; i < 1000; i++ {
			s.SetServingStatus(testService, healthpb.HealthCheckResponse_SERVING)
			time.Sleep(time.Microsecond)
		}
		wg.Done()
	}()
	go func() {
		time.Sleep(300 * time.Microsecond)
		s.Shutdown()
		wg.Done()
	}()
	wg.Wait()

	s.mu.Lock()
	status := s.statusMap[testService]
	s.mu.Unlock()
	assert.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, status, "status for %s is %v, want %v", testService, status, healthpb.HealthCheckResponse_NOT_SERVING)
}

func TestResume(t *testing.T) {
	s := NewServer()
	s.SetServingStatus(testService, healthpb.HealthCheckResponse_SERVING)
	s.Shutdown()
	status := s.statusMap[testService]
	assert.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, status, "status for %s is %v, want %v", testService, status, healthpb.HealthCheckResponse_NOT_SERVING)
	s.Resume()
	status = s.statusMap[testService]
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, status, "status for %s is %v, want %v", testService, status, healthpb.HealthCheckResponse_SERVING)
}
