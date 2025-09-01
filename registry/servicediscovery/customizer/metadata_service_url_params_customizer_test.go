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

package customizer

import (
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func TestMetadataServiceURLParamsMetadataCustomizer(t *testing.T) {

	msup := &metadataServiceURLParamsMetadataCustomizer{exceptKeys: gxset.NewSet()}
	assert.Equal(t, 0, msup.GetPriority())

	msup.Customize(createInstance())
}

func createInstance() registry.ServiceInstance {
	ins := &registry.DefaultServiceInstance{}
	return ins
}
