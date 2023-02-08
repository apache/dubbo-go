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
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type MockServiceNameMapping struct{}

func NewMockServiceNameMapping() *MockServiceNameMapping {
	return &MockServiceNameMapping{}
}

func (m *MockServiceNameMapping) Map(*common.URL) error {
	return nil
}

func (m *MockServiceNameMapping) Get(*common.URL, registry.MappingListener) (*gxset.HashSet, error) {
	panic("implement me")
}

func (m *MockServiceNameMapping) Remove(*common.URL) error {
	panic("implement me")
}
