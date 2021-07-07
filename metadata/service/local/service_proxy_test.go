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

package local

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func TestMetadataServiceProxy_GetExportedURLs(t *testing.T) {
	pxy := createPxy()
	assert.NotNil(t, pxy)
	res, err := pxy.GetExportedURLs(constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE)
	assert.Nil(t, err)
	assert.Len(t, res, 1)
}

// TestNewMetadataService: those methods are not implemented
// when we implement them, adding UT
func TestNewMetadataService(t *testing.T) {
	pxy := createPxy()
	_, err := pxy.ServiceName()
	assert.Nil(t, err)
	err = pxy.PublishServiceDefinition(&common.URL{})
	assert.Nil(t, err)
	_, err = pxy.GetServiceDefinition(constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE)
	assert.Nil(t, err)
	_, err = pxy.Version()
	assert.Nil(t, err)
	_, err = pxy.GetSubscribedURLs()
	assert.Nil(t, err)
	err = pxy.UnsubscribeURL(&common.URL{})
	assert.Nil(t, err)
	_, err = pxy.GetServiceDefinitionByServiceKey("any")
	assert.Nil(t, err)
	ok, err := pxy.ExportURL(&common.URL{})
	assert.False(t, ok)
	assert.NoError(t, err)
	ok, err = pxy.SubscribeURL(&common.URL{})
	assert.False(t, ok)
	assert.NoError(t, err)
	m := pxy.MethodMapper()
	assert.True(t, len(m) == 0)
	err = pxy.UnexportURL(&common.URL{})
	assert.NoError(t, err)
	ok, err = pxy.RefreshMetadata(constant.ANY_VALUE, constant.ANY_VALUE)
	assert.False(t, ok)
	assert.NoError(t, err)
}

func createPxy() service.MetadataService {
	extension.SetProtocol("mock", func() protocol.Protocol {
		return &mockProtocol{}
	})

	ins := &registry.DefaultServiceInstance{
		ID:          "test-id",
		ServiceName: "com.dubbo",
		Host:        "localhost",
		Port:        8080,
		Enable:      true,
		Healthy:     true,
		Metadata:    map[string]string{constant.METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME: `{"timeout":"10000", "protocol":"mock","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"20880"}`},
	}

	return extension.GetMetadataServiceProxyFactory("").GetProxy(ins)
}
