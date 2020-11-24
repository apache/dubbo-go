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

package random

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/registry"
)

func TestRandomServiceInstanceSelector_Select(t *testing.T) {
	selector := NewRandomServiceInstanceSelector()
	serviceInstances := []registry.ServiceInstance{
		&registry.DefaultServiceInstance{
			Id:          "1",
			ServiceName: "test1",
			Host:        "127.0.0.1:80",
			Port:        0,
			Enable:      false,
			Healthy:     false,
			Metadata:    nil,
		},
		&registry.DefaultServiceInstance{
			Id:          "2",
			ServiceName: "test2",
			Host:        "127.0.0.1:80",
			Port:        0,
			Enable:      false,
			Healthy:     false,
			Metadata:    nil,
		},
	}
	assert.NotNil(t, selector.Select(&common.URL{}, serviceInstances))
}
