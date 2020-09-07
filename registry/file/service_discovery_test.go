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

package file

import (
	"math/rand"
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/registry"
)

var (
	testName = "test"
)

func TestNewFileSystemServiceDiscoveryAndDestroy(t *testing.T) {
	prepareData()
	serviceDiscovery, err := newFileSystemServiceDiscovery(testName)
	assert.NoError(t, err)
	assert.NotNil(t, serviceDiscovery)
	defer serviceDiscovery.Destroy()
}

func TestCURDFileSystemServiceDiscovery(t *testing.T) {
	prepareData()
	serviceDiscovery, err := extension.GetServiceDiscovery(constant.FILE_KEY, testName)
	assert.NoError(t, err)
	md := make(map[string]string)

	rand.Seed(time.Now().Unix())
	serviceName := "service-name" + strconv.Itoa(rand.Intn(10000))
	md["t1"] = "test1"
	r1 := &registry.DefaultServiceInstance{
		Id:          "123456789",
		ServiceName: serviceName,
		Host:        "127.0.0.1",
		Port:        2233,
		Enable:      true,
		Healthy:     true,
		Metadata:    md,
	}
	err = serviceDiscovery.Register(r1)
	assert.NoError(t, err)

	instances := serviceDiscovery.GetInstances(r1.ServiceName)
	assert.Equal(t, 1, len(instances))
	assert.Equal(t, r1.Id, instances[0].GetId())
	assert.Equal(t, r1.ServiceName, instances[0].GetServiceName())
	assert.Equal(t, r1.Port, instances[0].GetPort())

	err = serviceDiscovery.Unregister(r1)
	assert.NoError(t, err)

	err = serviceDiscovery.Register(r1)

	defer serviceDiscovery.Destroy()
}

func prepareData() {
	config.GetBaseConfig().ServiceDiscoveries[testName] = &config.ServiceDiscoveryConfig{
		Protocol: "file",
	}
}
