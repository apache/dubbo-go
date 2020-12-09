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

package inmemory

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/metadata/service"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/registry"
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
	pxy.ServiceName()
	pxy.PublishServiceDefinition(&common.URL{})
	pxy.GetServiceDefinition(constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE)
	pxy.Version()
	pxy.GetSubscribedURLs()
	pxy.UnsubscribeURL(&common.URL{})
	pxy.GetServiceDefinitionByServiceKey("any")
	pxy.ExportURL(&common.URL{})
	pxy.SubscribeURL(&common.URL{})
	pxy.MethodMapper()
	pxy.UnexportURL(&common.URL{})
	pxy.RefreshMetadata(constant.ANY_VALUE, constant.ANY_VALUE)

}

func createPxy() service.MetadataService {
	extension.SetProtocol("mock", func() protocol.Protocol {
		return &mockProtocol{}
	})

	ins := &registry.DefaultServiceInstance{
		Id:          "test-id",
		ServiceName: "com.dubbo",
		Host:        "localhost",
		Port:        8080,
		Enable:      true,
		Healthy:     true,
		Metadata:    map[string]string{constant.METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME: `{"mock":{"timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"20880"}}`},
	}

	return extension.GetMetadataServiceProxyFactory(local).GetProxy(ins)
}
