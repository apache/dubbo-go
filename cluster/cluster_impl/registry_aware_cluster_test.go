package cluster_impl

import (
	"context"
	"fmt"
	"testing"
)
import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster/directory"
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	"github.com/dubbo/go-for-apache-dubbo/protocol/invocation"
)

func Test_RegAwareInvokeSuccess(t *testing.T) {

	regAwareCluster := NewRegistryAwareCluster()

	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, NewMockInvoker(url, 1))
	}

	staticDir := directory.NewStaticDirectory(invokers)
	clusterInvoker := regAwareCluster.Join(staticDir)
	result := clusterInvoker.Invoke(&invocation.RPCInvocation{})
	assert.NoError(t, result.Error())
	count = 0
}

func TestDestroy(t *testing.T) {
	regAwareCluster := NewRegistryAwareCluster()

	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, NewMockInvoker(url, 1))
	}

	staticDir := directory.NewStaticDirectory(invokers)
	clusterInvoker := regAwareCluster.Join(staticDir)
	assert.Equal(t, true, clusterInvoker.IsAvailable())
	result := clusterInvoker.Invoke(&invocation.RPCInvocation{})
	assert.NoError(t, result.Error())
	count = 0
	clusterInvoker.Destroy()
	assert.Equal(t, false, clusterInvoker.IsAvailable())

}
