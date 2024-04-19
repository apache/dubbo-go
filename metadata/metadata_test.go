package metadata

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
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
			assert.True(t, registryMetadataInfo[tt.args.registryId] != nil)
			meta := registryMetadataInfo[tt.args.registryId]
			meta.App = tt.args.url.GetParam(constant.ApplicationKey, "")
			meta.Tag = tt.args.url.GetParam(constant.ApplicationTagKey, "")
			assert.True(t, reflect.DeepEqual(meta.GetExportedServiceURLs()[0], tt.args.url))
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
			assert.True(t, registryMetadataInfo[tt.args.registryId] != nil)
			meta := registryMetadataInfo[tt.args.registryId]
			meta.App = tt.args.url.GetParam(constant.ApplicationKey, "")
			meta.Tag = tt.args.url.GetParam(constant.ApplicationTagKey, "")
			assert.True(t, reflect.DeepEqual(meta.GetSubscribedURLs()[0], tt.args.url))
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
