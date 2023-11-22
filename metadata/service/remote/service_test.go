package client

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
)

func TestCreateRpcClient(t *testing.T) {
	url, err := common.NewURL("dubbo://10.253.77.49:20881")
	if err != nil {
		t.Error(err)
	}
	url.SetParam(constant.SideKey, constant.Consumer)
	url.SetParam(constant.VersionKey, "1.0.0")
	url.SetParam(constant.InterfaceKey, constant.MetadataServiceName)
	url.SetParam(constant.GroupKey, "metrics-provider")
	rpcService, err := createRpcClient(url)
	if err != nil {
		t.Error(err)
	}
	metadataInfo, err := rpcService.GetMetadataInfo(context.Background(), "d7c522bd296066ecde5c32e631bcdbe9")
	if err != nil {
		t.Error(err)
	}
	assert.NotNil(t, metadataInfo)
}
