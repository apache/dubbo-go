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

package metadata

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
)

func createTestURL() *common.URL {
	return common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
		common.WithParamsValue(constant.ApplicationTagKey, "v1"),
	)
}

func TestAddService(t *testing.T) {
	registryId := "test-reg-add"
	url := createTestURL()

	AddService(registryId, url)

	assert.NotNil(t, registryMetadataInfo[registryId])
	meta := registryMetadataInfo[registryId]
	assert.Equal(t, url, meta.GetExportedServiceURLs()[0])
}

func TestAddSubscribeURL(t *testing.T) {
	registryId := "test-reg-sub"
	url := createTestURL()

	AddSubscribeURL(registryId, url)

	assert.NotNil(t, registryMetadataInfo[registryId])
	meta := registryMetadataInfo[registryId]
	assert.Equal(t, url, meta.GetSubscribedURLs()[0])
}

func TestGetMetadataInfo(t *testing.T) {
	registryId := "test-reg-get"
	expected := info.NewMetadataInfo("test-app", "v2")
	registryMetadataInfo[registryId] = expected

	assert.Equal(t, expected, GetMetadataInfo(registryId))
}

func TestGetMetadataService(t *testing.T) {
	assert.Equal(t, metadataService, GetMetadataService())
}
