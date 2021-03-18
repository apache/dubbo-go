package getty

import (
	. "github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol/dubbo/impl"
	"github.com/apache/dubbo-go/remoting"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestTCPPackageHandle(t *testing.T) {
	svr, url := InitTest(t)
	client := getClient(url)
	testDecodeTCPPackage(t, svr, client)
	svr.Stop()
}

func testDecodeTCPPackage(t *testing.T, svr *Server, client *Client) {
	request := buildTestRequest()
	pkgWriteHandler := NewRpcClientPackageHandler(client)
	pkgBytes, err := pkgWriteHandler.Write(nil, request)
	assert.NoError(t, err)
	pkgReadHandler := NewRpcServerPackageHandler(svr)
	_, pkgLen, err := pkgReadHandler.Read(nil, pkgBytes)
	assert.NoError(t, err)
	assert.Equal(t, pkgLen, len(pkgBytes))

	// simulate incomplete tcp package
	incompletePkgLen := len(pkgBytes) - 10
	assert.True(t, incompletePkgLen >= impl.HEADER_LENGTH, "header buffer too short")
	incompletePkg := pkgBytes[0 : incompletePkgLen-1]
	pkg, pkgLen, err := pkgReadHandler.Read(nil, incompletePkg)
	assert.NoError(t, err)
	assert.Equal(t, pkg, nil)
	assert.Equal(t, pkgLen, 0)
}

func buildTestRequest() *remoting.Request {
	request := remoting.NewRequest("2.0.2")
	up := &UserProvider{}
	invocation := createInvocation("GetUser", nil, nil, []interface{}{[]interface{}{"1", "username"}},
		[]reflect.Value{reflect.ValueOf([]interface{}{"1", "username"}), reflect.ValueOf(up)})
	attachment := map[string]string{INTERFACE_KEY: "com.ikurento.user.UserProvider",
		PATH_KEY:    "UserProvider",
		VERSION_KEY: "1.0.0",
	}
	setAttachment(invocation, attachment)
	request.Data = invocation
	request.Event = false
	request.TwoWay = false
	return request
}
