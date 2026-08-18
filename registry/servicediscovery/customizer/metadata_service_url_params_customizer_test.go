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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func TestMetadataServiceURLParamsMetadataCustomizerGetPriority(t *testing.T) {
	msup := &metadataServiceURLParamsMetadataCustomizer{exceptKeys: gxset.NewSet()}
	assert.Equal(t, 0, msup.GetPriority())
}

// TestMetadataServiceURLParamsCustomizeNilURL verifies that when the metadata
// service URL is not exported (nil), Customize writes nothing into metadata.
func TestMetadataServiceURLParamsCustomizeNilURL(t *testing.T) {
	msup := &metadataServiceURLParamsMetadataCustomizer{exceptKeys: gxset.NewSet()}
	ins := &registry.DefaultServiceInstance{}
	msup.Customize(ins)
	_, ok := ins.GetMetadata()[constant.MetadataServiceURLParamsPropertyName]
	assert.False(t, ok, "nothing should be written when the metadata service URL is nil")
}

func TestConvertToParams(t *testing.T) {
	msup := &metadataServiceURLParamsMetadataCustomizer{exceptKeys: gxset.NewSet()}

	u := common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithPort("20880"),
		common.WithParamsValue(constant.TimeoutKey, "3000"), // in IncludeKeys, should be kept
		common.WithParamsValue(constant.PathKey, "/path"),   // in IncludeKeys, should be kept
		common.WithParamsValue(constant.VersionKey, ""),     // empty value, should be dropped
		common.WithParamsValue("custom.arbitrary.key", "x"), // not in IncludeKeys, should be dropped
	)

	ps := msup.convertToParams(u)

	assert.Equal(t, "3000", ps[constant.TimeoutKey])
	assert.Equal(t, "/path", ps[constant.PathKey])
	// port/protocol are always appended even if absent from URL params
	assert.Equal(t, "dubbo", ps[constant.ProtocolKey])
	assert.Equal(t, "20880", ps[constant.PortKey])
	// empty values are dropped
	_, ok := ps[constant.VersionKey]
	assert.False(t, ok, "empty value param should be dropped")
	// keys outside info.IncludeKeys are dropped
	_, ok = ps["custom.arbitrary.key"]
	assert.False(t, ok, "param not in IncludeKeys should be dropped")
}

func TestConvertToParamsAlwaysAppendsPortAndProtocol(t *testing.T) {
	msup := &metadataServiceURLParamsMetadataCustomizer{exceptKeys: gxset.NewSet()}

	// URL without explicit port/protocol params
	u := common.NewURLWithOptions(
		common.WithProtocol("tri"),
		common.WithPort("50051"),
	)

	ps := msup.convertToParams(u)

	assert.Equal(t, "tri", ps[constant.ProtocolKey])
	assert.Equal(t, "50051", ps[constant.PortKey])
}
