package inmemory

import (
	"fmt"
	"github.com/apache/dubbo-go/common"
	"github.com/bmizerany/assert"
	"testing"
)

func TestMetadataService(t *testing.T) {
	mts := NewMetadataService()
	serviceName := "com.ikurento.user.UserProvider"
	group := "group1"
	version := "0.0.1"
	protocol := "dubbo"
	u, _ := common.NewURL(fmt.Sprintf("%v://127.0.0.1:20000/com.ikurento.user.UserProvider?anyhost=true&"+
		"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
		"environment=dev&interface=%v&ip=192.168.56.1&methods=GetUser&"+
		"module=dubbogo+user-info+server&org=ikurento.com&owner=ZX&pid=1447&revision=0.0.1&"+
		"side=provider&timeout=3000&timestamp=1556509797245&group=%v&version=%v", protocol, serviceName, group, version))
	mts.ExportURL(u)
	sets := mts.GetExportedURLs(serviceName, group, version, protocol)
	assert.Equal(t, 1, sets.Size())
	mts.SubscribeURL(u)

	mts.SubscribeURL(u)
	sets2 := mts.GetSubscribedURLs()
	assert.Equal(t, 1, sets2.Size())

	mts.UnexportURL(u)
	sets11 := mts.GetExportedURLs(serviceName, group, version, protocol)
	assert.Equal(t, 0, sets11.Size())

	mts.UnsubscribeURL(u)
	sets22 := mts.GetSubscribedURLs()
	assert.Equal(t, 0, sets22.Size())
}
