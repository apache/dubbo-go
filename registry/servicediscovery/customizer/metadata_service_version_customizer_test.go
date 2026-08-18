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
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func TestMetadtaServiceVersionCustomizerGetPriority(t *testing.T) {
	p := &MetadtaServiceVersionCustomizer{}
	assert.Equal(t, 0, p.GetPriority())
}

// TestMetadtaServiceVersionCustomizerTriProtocol verifies that tri writes v2.
func TestMetadtaServiceVersionCustomizerTriProtocol(t *testing.T) {
	p := &MetadtaServiceVersionCustomizer{}
	ins := newVersionInstance(`{"protocol":"tri","port":"20880"}`)
	p.Customize(ins)
	assert.Equal(t, constant.MetadataServiceV2Version, ins.GetMetadata()[constant.MetadataVersion])
}

// TestMetadtaServiceVersionCustomizerDubboProtocol verifies that dubbo writes v1.
func TestMetadtaServiceVersionCustomizerDubboProtocol(t *testing.T) {
	p := &MetadtaServiceVersionCustomizer{}
	ins := newVersionInstance(`{"protocol":"dubbo","port":"20880"}`)
	p.Customize(ins)
	assert.Equal(t, constant.MetadataServiceV1Version, ins.GetMetadata()[constant.MetadataVersion])
}

// TestMetadtaServiceVersionCustomizerUnknownProtocol verifies that any protocol
// other than tri (including an absent one) falls back to v1.
func TestMetadtaServiceVersionCustomizerUnknownProtocol(t *testing.T) {
	p := &MetadtaServiceVersionCustomizer{}
	ins := newVersionInstance(`{"port":"20880"}`)
	p.Customize(ins)
	assert.Equal(t, constant.MetadataServiceV1Version, ins.GetMetadata()[constant.MetadataVersion])
}

// TestMetadtaServiceVersionCustomizerNonLocalStorage verifies that the version
// is not written when the metadata storage type is not local.
func TestMetadtaServiceVersionCustomizerNonLocalStorage(t *testing.T) {
	p := &MetadtaServiceVersionCustomizer{}
	ins := &registry.DefaultServiceInstance{
		Metadata: map[string]string{
			constant.MetadataStorageTypePropertyName:      "remote",
			constant.MetadataServiceURLParamsPropertyName: `{"protocol":"tri","port":"20880"}`,
		},
	}
	p.Customize(ins)
	_, ok := ins.GetMetadata()[constant.MetadataVersion]
	assert.False(t, ok, "version should not be written for non-local storage type")
}

// TestMetadtaServiceVersionCustomizerInvalidJSON verifies that an unparsable
// params JSON leaves the metadata unchanged.
func TestMetadtaServiceVersionCustomizerInvalidJSON(t *testing.T) {
	p := &MetadtaServiceVersionCustomizer{}
	ins := newVersionInstance("not-a-json")
	p.Customize(ins)
	_, ok := ins.GetMetadata()[constant.MetadataVersion]
	assert.False(t, ok, "version should not be written when the params JSON is invalid")
}

// newVersionInstance creates an instance with local storage type and the given
// metadata service url params JSON.
func newVersionInstance(paramsJSON string) registry.ServiceInstance {
	return &registry.DefaultServiceInstance{
		Metadata: map[string]string{
			constant.MetadataStorageTypePropertyName:      constant.DefaultMetadataStorageType,
			constant.MetadataServiceURLParamsPropertyName: paramsJSON,
		},
	}
}
