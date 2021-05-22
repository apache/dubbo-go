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
	"dubbo.apache.org/dubbo-go/v3/registry/servicediscovery/instance"
)

var serviceInstanceSelectorMappings = make(map[string]func() instance.ServiceInstanceSelector, 2)

// nolint
func SetServiceInstanceSelector(name string, f func() instance.ServiceInstanceSelector) {
	serviceInstanceSelectorMappings[name] = f
}

// GetServiceInstanceSelector will create an instance
// it will panic if selector with the @name not found
func GetServiceInstanceSelector(name string) (instance.ServiceInstanceSelector, error) {
	serviceInstanceSelector, ok := serviceInstanceSelectorMappings[name]
	if !ok {
		return nil, perrors.New("Could not find service instance selector with" +
			"name:" + name)
	}
	return serviceInstanceSelector(), nil
}
