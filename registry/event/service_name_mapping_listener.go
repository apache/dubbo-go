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

package event

import (
	"reflect"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/observer"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
)

func init() {
	extension.AddEventListener(GetServiceNameMappingListener)
}

// serviceNameMappingListener listen to service name mapping event
// usually it means that we exported some service
// it's a singleton
type serviceNameMappingListener struct {
	nameMapping mapping.ServiceNameMapping
}

// GetPriority return 3, which ensure that this listener will be invoked after log listener
func (s *serviceNameMappingListener) GetPriority() int {
	return 3
}

// OnEvent only handle ServiceConfigExportedEvent
func (s *serviceNameMappingListener) OnEvent(e observer.Event) error {
	if ex, ok := e.(*ServiceConfigExportedEvent); ok {
		sc := ex.ServiceConfig
		urls := sc.GetExportedUrls()

		for _, u := range urls {
			err := s.nameMapping.Map(u.GetParam(constant.INTERFACE_KEY, ""),
				u.GetParam(constant.GROUP_KEY, ""),
				u.GetParam(constant.Version, ""),
				u.Protocol)
			if err != nil {
				return perrors.WithMessage(err, "could not map the service: "+u.String())
			}
		}
	}
	return nil
}

// GetEventType return ServiceConfigExportedEvent
func (s *serviceNameMappingListener) GetEventType() reflect.Type {
	return reflect.TypeOf(&ServiceConfigExportedEvent{})
}

var (
	serviceNameMappingListenerInstance *serviceNameMappingListener
	serviceNameMappingListenerOnce     sync.Once
)

// GetServiceNameMappingListener returns an instance
func GetServiceNameMappingListener() observer.EventListener {
	serviceNameMappingListenerOnce.Do(func() {
		serviceNameMappingListenerInstance = &serviceNameMappingListener{
			nameMapping: extension.GetGlobalServiceNameMapping(),
		}
	})
	return serviceNameMappingListenerInstance
}
