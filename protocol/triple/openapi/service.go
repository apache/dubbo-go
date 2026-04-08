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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/openapi/model"
)

type OpenAPIRequest struct {
	Group string
}

// serviceKey is a composite key that uniquely identifies a registered service
// at the OpenAPI document level. Using only interfaceName as the key would cause
// collisions when the same interface is exported into multiple OpenAPI groups,
// or appears under different Dubbo group/version combinations.
type serviceKey struct {
	interfaceName string
	group         string
	dubboGroup    string
	dubboVersion  string
}

// serviceInfo holds a snapshot of the data needed from common.ServiceInfo
// for OpenAPI generation. By copying only the necessary fields at registration
// time, we decouple from the original ServiceInfo and avoid side-effects if
// the caller later mutates its Methods or Meta.
type serviceInfo struct {
	Methods []serviceMethodInfo
}

// serviceMethodInfo holds a snapshot of the data needed from common.MethodInfo
// for OpenAPI generation.
type serviceMethodInfo struct {
	Name        string
	ReqInitFunc func() any
	Meta        map[string]any
}

type DefaultService struct {
	config      *global.OpenAPIConfig
	defResolver *DefinitionResolver
	encoder     *Encoder

	mu       sync.RWMutex
	openAPIs map[string]*model.OpenAPI
	services map[serviceKey]*serviceInfo
}

func NewDefaultService(cfg *global.OpenAPIConfig) *DefaultService {
	if cfg == nil {
		cfg = global.DefaultOpenAPIConfig()
	}
	s := &DefaultService{
		config:   cfg,
		services: make(map[serviceKey]*serviceInfo),
		openAPIs: make(map[string]*model.OpenAPI),
	}
	s.defResolver = NewDefinitionResolver(s.config)
	s.encoder = NewEncoder()
	return s
}

func (s *DefaultService) RegisterService(interfaceName string, info *common.ServiceInfo, openapiGroup string, dubboGroup string, dubboVersion string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if openapiGroup == "" {
		openapiGroup = constant.OpenAPIDefaultGroup
	}
	s.services[serviceKey{
		interfaceName: interfaceName,
		group:         openapiGroup,
		dubboGroup:    dubboGroup,
		dubboVersion:  dubboVersion,
	}] = snapshotServiceInfo(info)
	s.openAPIs = nil
}

func (s *DefaultService) GetOpenAPI(req *OpenAPIRequest) *model.OpenAPI {
	if req == nil {
		req = &OpenAPIRequest{}
	}

	openAPIs := s.getOpenAPIs()

	group := req.Group
	if group == "" {
		group = constant.OpenAPIDefaultGroup
	}

	if openAPI, ok := openAPIs[group]; ok {
		return openAPI
	}

	// Group not found — return an empty OpenAPI document instead of the merged spec.
	empty := model.NewOpenAPI()
	empty.Info = &model.Info{
		Title:   s.config.InfoTitle,
		Version: s.config.InfoVersion,
	}
	return empty
}

func (s *DefaultService) GetOpenAPIGroups() []string {
	openAPIs := s.getOpenAPIs()
	groups := []string{constant.OpenAPIDefaultGroup}
	seen := map[string]bool{constant.OpenAPIDefaultGroup: true}

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

	for key, info := range s.services {
		openAPI := s.defResolver.Resolve(key.interfaceName, info)
		if openAPI != nil {
			group := key.group
			if group == "" {
				group = constant.OpenAPIDefaultGroup
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

func (s *DefaultService) mergeOpenAPIs(openAPIs map[string]*model.OpenAPI) *model.OpenAPI {
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

// snapshotServiceInfo creates a defensive copy of only the fields from
// common.ServiceInfo that are needed for OpenAPI generation.
func snapshotServiceInfo(info *common.ServiceInfo) *serviceInfo {
	if info == nil {
		return nil
	}
	si := &serviceInfo{}
	if len(info.Methods) > 0 {
		si.Methods = make([]serviceMethodInfo, len(info.Methods))
		for i, m := range info.Methods {
			si.Methods[i] = serviceMethodInfo{
				Name:        m.Name,
				ReqInitFunc: m.ReqInitFunc,
			}
			if len(m.Meta) > 0 {
				si.Methods[i].Meta = make(map[string]any, len(m.Meta))
				for k, v := range m.Meta {
					si.Methods[i].Meta[k] = v
				}
			}
		}
	}
	return si
}
