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

package info

import (
	"encoding/json"
	"testing"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestMetadataInfoAddService(t *testing.T) {
	metadataInfo := &MetadataInfo{
		Services:              make(map[string]*ServiceInfo),
		exportedServiceURLs:   make(map[string][]*common.URL),
		subscribedServiceURLs: make(map[string][]*common.URL),
	}

	url, _ := common.NewURL("dubbo://127.0.0.1:20000?application=foo&category=providers&check=false&dubbo=dubbo-go+v1.5.0&interface=com.foo.Bar&methods=GetPetByID%2CGetPetTypes&organization=Apache&owner=foo&revision=1.0.0&side=provider&version=1.0.0")
	metadataInfo.AddService(url)
	assert.True(t, len(metadataInfo.Services) > 0)
	assert.True(t, len(metadataInfo.exportedServiceURLs) > 0)

	metadataInfo.RemoveService(url)
	assert.True(t, len(metadataInfo.Services) == 0)
	assert.True(t, len(metadataInfo.exportedServiceURLs) == 0)
}

func TestHessian(t *testing.T) {
	metadataInfo := &MetadataInfo{
		App:                   "test",
		Revision:              "1",
		Services:              make(map[string]*ServiceInfo),
		exportedServiceURLs:   make(map[string][]*common.URL),
		subscribedServiceURLs: make(map[string][]*common.URL),
	}
	metadataInfo.Services["1"] = NewServiceInfo("dubbo.io", "default", "1.0.0", "dubbo", "", make(map[string]string))
	e := hessian.NewEncoder()
	err := e.Encode(metadataInfo)
	assert.Nil(t, err)
	obj, err := hessian.NewDecoder(e.Buffer()).Decode()
	assert.Nil(t, err)
	objJson, _ := json.Marshal(obj)
	metaJson, _ := json.Marshal(metadataInfo)
	assert.Equal(t, objJson, metaJson)
}
