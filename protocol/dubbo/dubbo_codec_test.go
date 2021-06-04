package dubbo

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"github.com/stretchr/testify/assert"
	"testing"
)

var defaultDubboCodec = &DubboCodec{}

func TestDubboCodec(t *testing.T) {
	t.Run("url", testUrl)
	t.Run("metadataInfo", testMetadataInfo)
}

func testUrl(t *testing.T) {
	// mock response
	rsp := remoting.NewResponse(1, "")
	// mock URL
	url, err := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")
	assert.NoError(t, err)
	// mock RPCResult
	rpcResult := protocol.RPCResult{Rest: url}
	rsp.Result = rpcResult
	// encode response
	buf, err := defaultDubboCodec.EncodeResponse(rsp)
	assert.NoError(t, err)
	// decode response
	// TODO: check the rest of return values
	_, _, err = defaultDubboCodec.Decode(buf.Bytes())
	assert.NoError(t, err)
}

func testMetadataInfo(t *testing.T) {
	// mock response
	rsp := remoting.NewResponse(1, "")
	url, err := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")
	assert.NoError(t, err)
	serviceInfoMap := map[string]*common.ServiceInfo{
		"serviceInfo1": common.NewServiceInfoWithURL(url),
	}
	metadataInfo := common.NewMetadataInfo("test", "", serviceInfoMap)
	// mock RPCResult
	rpcResult := protocol.RPCResult{Rest: metadataInfo}
	rsp.Result = rpcResult
	// encode response
	buf, err := defaultDubboCodec.EncodeResponse(rsp)
	assert.NoError(t, err)
	// decode response
	// TODO: check the rest of return values
	_, _, err = defaultDubboCodec.Decode(buf.Bytes())
	assert.NoError(t, err)
}
