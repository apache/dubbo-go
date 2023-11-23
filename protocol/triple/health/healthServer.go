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
	"context"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dubbogo/grpc-go/codes"
	"github.com/dubbogo/grpc-go/status"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/health/triple_health"
	"dubbo.apache.org/dubbo-go/v3/server"
)

type HealthTripleServer struct {
	mu sync.RWMutex
	// If shutdown is true, it's expected all serving status is NOT_SERVING, and
	// will stay in NOT_SERVING.
	shutdown bool
	// statusMap stores the serving status of the services this Server monitors.
	statusMap map[string]triple_health.HealthCheckResponse_ServingStatus
	updates   map[string]map[triple_health.Health_WatchServer]chan triple_health.HealthCheckResponse_ServingStatus
}

var healthServer *HealthTripleServer

func NewServer() *HealthTripleServer {
	return &HealthTripleServer{
		statusMap: map[string]triple_health.HealthCheckResponse_ServingStatus{"": triple_health.HealthCheckResponse_NOT_SERVING},
		updates:   make(map[string]map[triple_health.Health_WatchServer]chan triple_health.HealthCheckResponse_ServingStatus),
	}
}

func (srv *HealthTripleServer) Check(ctx context.Context, req *triple_health.HealthCheckRequest) (*triple_health.HealthCheckResponse, error) {
	srv.mu.RLock()
	defer srv.mu.RUnlock()
	if servingStatus, ok := srv.statusMap[req.Service]; ok {
		return &triple_health.HealthCheckResponse{
			Status: servingStatus,
		}, nil
	}
	return nil, status.Error(codes.NotFound, "unknown service")
}

func (srv *HealthTripleServer) Watch(ctx context.Context, request *triple_health.HealthCheckRequest, server triple_health.Health_WatchServer) error {
	service := request.Service
	// update channel is used for getting service status updates.
	update := make(chan triple_health.HealthCheckResponse_ServingStatus, 1)
	srv.mu.Lock()
	// Puts the initial status to the channel.
	if servingStatus, ok := srv.statusMap[service]; ok {
		update <- servingStatus
	} else {
		update <- triple_health.HealthCheckResponse_SERVICE_UNKNOWN
	}

	// Registers the update channel to the correct place in the updates map.
	if _, ok := srv.updates[service]; !ok {
		srv.updates[service] = make(map[triple_health.Health_WatchServer]chan triple_health.HealthCheckResponse_ServingStatus)
	}
	srv.updates[service][server] = update
	defer func() {
		srv.mu.Lock()
		delete(srv.updates[service], server)
		srv.mu.Unlock()
	}()
	srv.mu.Unlock()

	var lastSentStatus triple_health.HealthCheckResponse_ServingStatus = -1
	for {
		select {
		// Status updated. Sends the up-to-date status to the client.
		case servingStatus := <-update:
			if lastSentStatus == servingStatus {
				continue
			}
			lastSentStatus = servingStatus
			err := server.Send(&triple_health.HealthCheckResponse{Status: servingStatus})
			if err != nil {
				return status.Error(codes.Canceled, "Stream has ended.")
			}
		// Context done. Removes the update channel from the updates map.
		case <-ctx.Done():
			return status.Error(codes.Canceled, "Stream has ended.")
		}
	}
}

// SetServingStatus is called when need to reset the serving status of a service
// or insert a new service entry into the statusMap.
func (srv *HealthTripleServer) SetServingStatus(service string, servingStatus triple_health.HealthCheckResponse_ServingStatus) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.shutdown {
		logger.Infof("health: status changing for %s to %v is ignored because health service is shutdown", service, servingStatus)
		return
	}

	srv.setServingStatusLocked(service, servingStatus)
}

func (srv *HealthTripleServer) setServingStatusLocked(service string, servingStatus triple_health.HealthCheckResponse_ServingStatus) {
	srv.statusMap[service] = servingStatus
	for _, update := range srv.updates[service] {
		// Clears previous updates, that are not sent to the client, from the channel.
		// This can happen if the client is not reading and the server gets flow control limited.
		select {
		case <-update:
		default:
		}
		// Puts the most recent update to the channel.
		update <- servingStatus
	}
}

// Shutdown sets all serving status to NOT_SERVING, and configures the server to
// ignore all future status changes.
//
// This changes serving status for all services. To set status for a particular
// services, call SetServingStatus().
func (srv *HealthTripleServer) Shutdown() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.shutdown = true
	for service := range srv.statusMap {
		srv.setServingStatusLocked(service, triple_health.HealthCheckResponse_NOT_SERVING)
	}
}

// Resume sets all serving status to SERVING, and configures the server to
// accept all future status changes.
//
// This changes serving status for all services. To set status for a particular
// services, call SetServingStatus().
func (srv *HealthTripleServer) Resume() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.shutdown = false
	for service := range srv.statusMap {
		srv.setServingStatusLocked(service, triple_health.HealthCheckResponse_SERVING)
	}
}

func init() {
	healthServer = NewServer()
	server.SetProServices(&server.ServiceDefinition{
		Handler: healthServer,
		Info:    &triple_health.Health_ServiceInfo,
		Opts:    []server.ServiceOption{server.WithNotRegister()},
	})
}

func SetServingStatusServing(service string) {
	healthServer.SetServingStatus(service, triple_health.HealthCheckResponse_SERVING)
}

func SetServingStatusNotServing(service string) {
	healthServer.SetServingStatus(service, triple_health.HealthCheckResponse_NOT_SERVING)
}
