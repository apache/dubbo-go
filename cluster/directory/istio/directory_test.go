package istio

import (
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestStaticDirList(t *testing.T) {
	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}

	staticDir := NewDirectory(invokers)
	list := staticDir.List(&invocation.RPCInvocation{})

	assert.Len(t, list, 10)
}

func TestStaticDirDestroy(t *testing.T) {
	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}

	staticDir := NewDirectory(invokers)
	assert.Equal(t, true, staticDir.IsAvailable())
	staticDir.Destroy()
	assert.Equal(t, false, staticDir.IsAvailable())
}
