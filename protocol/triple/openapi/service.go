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
	"errors"
	"fmt"
	"maps"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/internal/httpbinding"
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
	config       *global.OpenAPIConfig
	defResolver  *DefinitionResolver
	encoder      *Encoder
	useHTTPRules bool

	mu         sync.RWMutex
	openAPIs   map[string]*model.OpenAPI
	services   map[serviceKey]*serviceInfo
	resolveErr error
}

func NewDefaultService(cfg *global.OpenAPIConfig, useHTTPRules ...bool) *DefaultService {
	if cfg == nil {
		cfg = global.DefaultOpenAPIConfig()
	}
	useRules := len(useHTTPRules) > 0 && useHTTPRules[0]
	s := &DefaultService{
		config:       cfg,
		useHTTPRules: useRules,
		services:     make(map[serviceKey]*serviceInfo),
		openAPIs:     make(map[string]*model.OpenAPI),
	}
	s.defResolver = NewDefinitionResolver(s.config, useRules)
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
	s.resolveErr = nil
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
	s.resolveErr = nil
	logger.Debug("[Triple][OpenAPI] OpenAPI documents refreshed")
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
	s.resolveErr = nil

	for key, info := range s.services {
		openAPI, err := s.defResolver.ResolveWithError(key.interfaceName, info)
		if err != nil {
			s.resolveErr = errors.Join(s.resolveErr, fmt.Errorf("resolve OpenAPI for %s: %w", key.interfaceName, err))
			continue
		}
		if openAPI != nil {
			group := key.group
			if group == "" {
				group = constant.OpenAPIDefaultGroup
			}
			openAPI.Group = group
			s.completeModel(openAPI, group)
			if existing, ok := result[group]; ok {
				if err := s.mergeOpenAPIWithError(existing, openAPI); err != nil {
					s.resolveErr = errors.Join(s.resolveErr, fmt.Errorf("merge OpenAPI group %q: %w", group, err))
				}
			} else {
				result[group] = openAPI
			}
		}
	}

	return result
}

// GetResolveError returns the most recent HTTP/OpenAPI resolution error. It
// forces lazy OpenAPI resolution so callers can distinguish an empty document
// from a document that could not be generated.
func (s *DefaultService) GetResolveError() error {
	_ = s.getOpenAPIs()
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.resolveErr
}

func (s *DefaultService) mergeOpenAPIs(openAPIs map[string]*model.OpenAPI) *model.OpenAPI {
	merged := model.NewOpenAPI()
	merged.Info = &model.Info{
		Title:       s.config.InfoTitle,
		Version:     s.config.InfoVersion,
		Description: s.config.InfoDescription,
	}

	for _, openAPI := range openAPIs {
		_ = s.mergeOpenAPIWithError(merged, openAPI)
	}

	return merged
}

func (s *DefaultService) completeModel(openAPI *model.OpenAPI, group string) {
	if openAPI.Info == nil {
		openAPI.Info = &model.Info{}
	}
	if openAPI.Info.Title == "" {
		openAPI.Info.Title = "Dubbo-go OpenAPI"
	}
	if openAPI.Info.Version == "" {
		openAPI.Info.Version = "1.0.0"
	}
	if len(openAPI.Paths) == 0 {
		return
	}
	if openAPI.ExternalDocs == nil {
		openAPI.ExternalDocs = &model.ExternalDocs{
			Description: "ReDoc",
			URL:         s.config.Path + "/redoc/index.html?group=" + group,
		}
	}
}

func (s *DefaultService) mergeOpenAPI(target, source *model.OpenAPI) {
	_ = s.mergeOpenAPIWithError(target, source)
}

// mergeOpenAPIWithError merges paths operation-by-operation. A path item can
// legitimately contain different methods from different services; replacing
// the whole item would silently drop one of those operations. A conflicting
// operation ID is rejected so the generated document never advertises an
// arbitrary winner.
func (s *DefaultService) mergeOpenAPIWithError(target, source *model.OpenAPI) error {
	if target == nil || source == nil {
		return nil
	}

	if source.Paths != nil {
		if target.Paths == nil {
			target.Paths = make(map[string]*model.PathItem)
		}
		// Compare the original Google path templates as well as the normalized
		// OpenAPI path keys. Variable names are intentionally retained in
		// OpenAPI parameters, but they must not hide an HTTP route collision
		// across services (/{name} and /{id} match the same URL shape).
		routeOperations := make(map[string]string)
		for path, targetItem := range target.Paths {
			if targetItem == nil {
				continue
			}
			for method, operation := range targetItem.GetOperations() {
				if operation == nil {
					continue
				}
				routeKey := httpbinding.CanonicalRouteKey(method, pathItemTemplate(targetItem, path))
				if existingID, exists := routeOperations[routeKey]; exists && existingID != operation.OperationId {
					return fmt.Errorf("HTTP route %q has conflicting operations %q and %q", routeKey, existingID, operation.OperationId)
				}
				routeOperations[routeKey] = operation.OperationId
			}
		}
		for path, sourceItem := range source.Paths {
			if sourceItem == nil {
				continue
			}
			for method, operation := range sourceItem.GetOperations() {
				if operation == nil {
					continue
				}
				routeKey := httpbinding.CanonicalRouteKey(method, pathItemTemplate(sourceItem, path))
				if existingID, exists := routeOperations[routeKey]; exists && existingID != operation.OperationId {
					return fmt.Errorf("HTTP route %q has conflicting operations %q and %q", routeKey, existingID, operation.OperationId)
				}
				routeOperations[routeKey] = operation.OperationId
			}
		}
		// Validate all operation conflicts before mutating target so a failed
		// merge cannot leave a partially updated document behind.
		for path, sourceItem := range source.Paths {
			if sourceItem == nil {
				continue
			}
			targetItem := target.Paths[path]
			if targetItem == nil {
				continue
			}
			for method, sourceOperation := range sourceItem.GetOperations() {
				if sourceOperation == nil {
					continue
				}
				if existingOperation := targetItem.GetOperation(method); existingOperation != nil && existingOperation.OperationId != sourceOperation.OperationId {
					return fmt.Errorf("path %q has conflicting %s operations %q and %q", path, method, existingOperation.OperationId, sourceOperation.OperationId)
				}
			}
		}
		for path, sourceItem := range source.Paths {
			if sourceItem == nil {
				continue
			}
			targetItem := target.Paths[path]
			if targetItem == nil {
				target.Paths[path] = sourceItem
				continue
			}
			for method, sourceOperation := range sourceItem.GetOperations() {
				if sourceOperation == nil {
					continue
				}
				if existingOperation := targetItem.GetOperation(method); existingOperation != nil {
					continue
				}
				targetItem.SetOperation(method, sourceOperation)
			}
			if targetItem.Extensions == nil && sourceItem.Extensions != nil {
				targetItem.Extensions = maps.Clone(sourceItem.Extensions)
			}
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
	return nil
}

func pathItemTemplate(item *model.PathItem, fallback string) string {
	if item != nil && item.Extensions != nil {
		if template, ok := item.Extensions["x-google-path-template"].(string); ok && template != "" {
			return template
		}
	}
	return fallback
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
				maps.Copy(si.Methods[i].Meta, m.Meta)
			}
		}
	}
	return si
}
