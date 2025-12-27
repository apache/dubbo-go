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

func TestAddService(t *testing.T) {
	type args struct {
		registryId string
		url        *common.URL
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "add",
			args: args{
				registryId: "reg1",
				url: common.NewURLWithOptions(
					common.WithProtocol("dubbo"),
					common.WithParamsValue(constant.ApplicationKey, "dubbo"),
					common.WithParamsValue(constant.ApplicationTagKey, "v1"),
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddService(tt.args.registryId, tt.args.url)
			assert.NotNil(t, registryMetadataInfo[tt.args.registryId])
			meta := registryMetadataInfo[tt.args.registryId]
			meta.App = tt.args.url.GetParam(constant.ApplicationKey, "")
			meta.Tag = tt.args.url.GetParam(constant.ApplicationTagKey, "")
			assert.Equal(t, tt.args.url, meta.GetExportedServiceURLs()[0])
		})
	}
}

func TestAddSubscribeURL(t *testing.T) {
	type args struct {
		registryId string
		url        *common.URL
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "addSub",
			args: args{
				registryId: "reg2",
				url: common.NewURLWithOptions(
					common.WithProtocol("dubbo"),
					common.WithParamsValue(constant.ApplicationKey, "dubbo"),
					common.WithParamsValue(constant.ApplicationTagKey, "v1"),
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddSubscribeURL(tt.args.registryId, tt.args.url)
			assert.NotNil(t, registryMetadataInfo[tt.args.registryId])
			meta := registryMetadataInfo[tt.args.registryId]
			meta.App = tt.args.url.GetParam(constant.ApplicationKey, "")
			meta.Tag = tt.args.url.GetParam(constant.ApplicationTagKey, "")
			assert.Equal(t, tt.args.url, meta.GetSubscribedURLs()[0])
		})
	}
}

func TestGetMetadataInfo(t *testing.T) {
	type args struct {
		registryId string
	}
	tests := []struct {
		name string
		args args
		want *info.MetadataInfo
	}{
		{
			name: "get",
			args: args{
				registryId: "reg3",
			},
			want: info.NewMetadataInfo("dubbo", "v2"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registryMetadataInfo[tt.args.registryId] = tt.want
			assert.Equalf(t, tt.want, GetMetadataInfo(tt.args.registryId), "GetMetadataInfo(%v)", tt.args.registryId)
		})
	}
}

func TestGetMetadataService(t *testing.T) {
	tests := []struct {
		name string
		want MetadataService
	}{
		{
			name: "getMetadataService",
			want: metadataService,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GetMetadataService(), "GetMetadataService()")
		})
	}
}
