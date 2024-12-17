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

package mapping

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// ServiceNameMapping  is the interface which trys to build the mapping between application-level service and interface-level service.
type ServiceNameMapping interface {
	// Map the service to this application-level service, store the mapping
	Map(url *common.URL) error
	// Get the application-level services which are mapped to this service
	Get(url *common.URL, listener MappingListener) (*gxset.HashSet, error)
	// Remove the mapping between application-level service and interface-level service
	Remove(url *common.URL) error
}

type MappingListener interface {
	OnEvent(e observer.Event) error
	Stop()
}
