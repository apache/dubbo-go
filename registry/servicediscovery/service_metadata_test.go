package servicediscovery

import (
	"context"
	"fmt"
	"testing"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/info"
)

func TestCreateRpcClientByUrl(t *testing.T) {
	url, err := common.NewURL("dubbo://10.253.77.49:61086?group=dubbo.io&interface=org.apache.dubbo.metadata.MetadataService&side=consumer&version=1.0.0")
	service, destory := createRpcClientByUrl(url)
	defer destory()
	meta, err := service.GetMetadataInfo(context.Background(), "4647984502")
	if err != nil {
		logger.Error(err)
	}
	fmt.Println(meta)
}
