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

package channel

import (
	"sync"

	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

type ResponseInfo struct {
	ResponseNonce string
	VersionInfo   string
	ResourceNames []string
}

type ApiStore struct {
	mutex     sync.Mutex
	responses map[string]*ResponseInfo
	rdsMutex  sync.Mutex
}

const (
	EnvoyListener = resource.ListenerType
	EnvoyCluster  = resource.ClusterType
	EnvoyEndpoint = resource.EndpointType
	EnvoyRoute    = resource.RouteType
)

func NewApiStore() *ApiStore {
	return &ApiStore{
		responses: map[string]*ResponseInfo{},
	}
}

func (s *ApiStore) Store(typeUrl string, info *ResponseInfo) {
	s.mutex.Lock()
	s.responses[typeUrl] = info
	s.mutex.Unlock()
}

func (s *ApiStore) SetResourceNames(typeUrl string, ResourceNames []string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	info, ok := s.responses[typeUrl]
	if ok {
		info.ResourceNames = ResourceNames
		s.responses[typeUrl] = info
		return
	}
	s.responses[typeUrl] = &ResponseInfo{
		ResourceNames: ResourceNames,
	}
}

func (s *ApiStore) Find(typeUrl string) *ResponseInfo {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	info, ok := s.responses[typeUrl]
	if !ok {
		return &ResponseInfo{
			ResourceNames: []string{},
		}
	}
	return info
}
