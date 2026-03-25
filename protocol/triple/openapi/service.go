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

package openapi

import (
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/openapi/model"
)

type OpenAPIRequest struct {
	Group  string
	Format string
}

type Service interface {
	GetOpenAPI(req *OpenAPIRequest) *model.OpenAPI
	GetOpenAPIGroups() []string
	Refresh()
}

type DefaultService struct {
	config      *global.OpenAPIConfig
	defResolver *DefinitionResolver
	encoder     *Encoder

	mu       sync.RWMutex
	openAPIs map[string]*model.OpenAPI
	services map[string]*common.ServiceInfo
	groups   map[string]string
}

func NewDefaultService(cfg *global.OpenAPIConfig) *DefaultService {
	if cfg == nil {
		cfg = global.DefaultOpenAPIConfig()
	}
	s := &DefaultService{
		config:   cfg,
		services: make(map[string]*common.ServiceInfo),
		openAPIs: make(map[string]*model.OpenAPI),
		groups:   make(map[string]string),
	}
	s.defResolver = NewDefinitionResolver(s.config, nil)
	s.encoder = NewEncoder()
	return s
}

func (s *DefaultService) RegisterService(interfaceName string, info *common.ServiceInfo, group string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[interfaceName] = info
	if group != "" {
		s.groups[interfaceName] = group
	}
	s.openAPIs = nil
}

func (s *DefaultService) GetOpenAPI(req *OpenAPIRequest) *model.OpenAPI {
	if req == nil {
		req = &OpenAPIRequest{}
	}

	openAPIs := s.getOpenAPIs()

	group := req.Group
	if group == "" {
		group = DefaultGroup
	}

	if openAPI, ok := openAPIs[group]; ok {
		return openAPI
	}

	merged := s.mergeOpenAPIs(openAPIs, req)
	return merged
}

func (s *DefaultService) GetOpenAPIGroups() []string {
	openAPIs := s.getOpenAPIs()
	groups := []string{DefaultGroup}
	seen := map[string]bool{DefaultGroup: true}

	for _, openAPI := range openAPIs {
		if openAPI.Group != "" && !seen[openAPI.Group] {
			groups = append(groups, openAPI.Group)
			seen[openAPI.Group] = true
		}
	}
	return groups
}

func (s *DefaultService) Refresh() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.openAPIs = nil
	logger.Debug("OpenAPI documents refreshed")
}

func (s *DefaultService) getOpenAPIs() map[string]*model.OpenAPI {
	s.mu.RLock()
	if s.openAPIs != nil {
		defer s.mu.RUnlock()
		return s.openAPIs
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.openAPIs != nil {
		return s.openAPIs
	}

	s.openAPIs = s.resolveOpenAPIs()
	return s.openAPIs
}

func (s *DefaultService) resolveOpenAPIs() map[string]*model.OpenAPI {
	result := make(map[string]*model.OpenAPI)

	for interfaceName, info := range s.services {
		openAPI := s.defResolver.Resolve(interfaceName, info)
		if openAPI != nil {
			group := s.groups[interfaceName]
			if group == "" {
				group = DefaultGroup
			}
			openAPI.Group = group
			if existing, ok := result[group]; ok {
				s.mergeOpenAPI(existing, openAPI)
			} else {
				result[group] = openAPI
			}
		}
	}

	return result
}

func (s *DefaultService) mergeOpenAPIs(openAPIs map[string]*model.OpenAPI, _ *OpenAPIRequest) *model.OpenAPI {
	merged := model.NewOpenAPI()
	merged.Info = &model.Info{
		Title:       s.config.InfoTitle,
		Version:     s.config.InfoVersion,
		Description: s.config.InfoDescription,
	}

	for _, openAPI := range openAPIs {
		s.mergeOpenAPI(merged, openAPI)
	}

	return merged
}

func (s *DefaultService) mergeOpenAPI(target, source *model.OpenAPI) {
	if source.Paths != nil {
		if target.Paths == nil {
			target.Paths = make(map[string]*model.PathItem)
		}
		for path, item := range source.Paths {
			target.Paths[path] = item
		}
	}

	if source.Components != nil {
		if target.Components == nil {
			target.Components = model.NewComponents()
		}
		for name, schema := range source.Components.Schemas {
			target.Components.AddSchema(name, schema)
		}
	}
}

func (s *DefaultService) GetEncoder() *Encoder {
	return s.encoder
}

func (s *DefaultService) GetConfig() *global.OpenAPIConfig {
	return s.config
}
