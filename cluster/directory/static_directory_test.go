package directory

import (
	"context"
	"fmt"
	"github.com/dubbo/go-for-apache-dubbo/protocol/invocation"
	"github.com/stretchr/testify/assert"
	"testing"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

func Test_StaticDirList(t *testing.T) {
	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}

	staticDir := NewStaticDirectory(invokers)
	assert.Len(t, staticDir.List(&invocation.RPCInvocation{}), 10)
}

func Test_StaticDirDestroy(t *testing.T) {
	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, protocol.NewBaseInvoker(url))
	}

	staticDir := NewStaticDirectory(invokers)
	assert.Equal(t, true, staticDir.IsAvailable())
	staticDir.Destroy()
	assert.Equal(t, false, staticDir.IsAvailable())
}
