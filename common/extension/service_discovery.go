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

package extension

import (
	perrors "github.com/pkg/errors"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/registry"
)

var (
	discoveryCreatorMap = make(map[string]func(url *common.URL) (registry.ServiceDiscovery, error), 4)
)

// SetServiceDiscovery will store the @creator and @name
func SetServiceDiscovery(name string, creator func(_ *common.URL) (registry.ServiceDiscovery, error)) {
	discoveryCreatorMap[name] = creator
}

// GetServiceDiscovery will return the registry.ServiceDiscovery
// if not found, or initialize instance failed, it will return error.
func GetServiceDiscovery(name string, url *common.URL) (registry.ServiceDiscovery, error) {
	creator, ok := discoveryCreatorMap[name]
	if !ok {
		return nil, perrors.New("Could not find the service discovery with name: " + name)
	}
	return creator(url)
}
