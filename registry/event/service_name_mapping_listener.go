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

	perrors "github.com/pkg/errors"

	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/observer"
	"github.com/apache/dubbo-go/metadata/mapping"
)

func init() {
	extension.AddEventListener(&serviceNameMappingListener{
		nameMapping: extension.GetGlobalServiceNameMapping(),
	})
}

type serviceNameMappingListener struct {
	nameMapping mapping.ServiceNameMapping
}

// GetPriority return 3, which ensure that this listener will be invoked after log listener
func (s *serviceNameMappingListener) GetPriority() int {
	return 3
}

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

func (s *serviceNameMappingListener) GetEventType() reflect.Type {
	return reflect.TypeOf(&ServiceConfigExportedEvent{})
}
