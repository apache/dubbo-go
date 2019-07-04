package configurator

import (
	"context"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_configureVerison2p6(t *testing.T) {
	url, err := common.NewURL(context.Background(), "override://0.0.0.0:0/com.xxx.mock.userProvider?group=1&version=1&cluster=failfast&application=BDTService")
	assert.NoError(t, err)
	configurator := extension.GetConfigurator("override", &url)
	assert.Equal(t, "override", configurator.GetUrl().Protocol)

	providerUrl, err := common.NewURL(context.Background(), "jsonrpc://127.0.0.1:20001/com.ikurento.user.UserProvider?anyhost=true&app.version=0.0.1&application=BDTService&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&group=&interface=com.ikurento.user.UserProvider&ip=10.32.20.124&loadbalance=random&methods.GetUser.loadbalance=random&methods.GetUser.retries=1&methods.GetUser.weight=0&module=dubbogo+user-info+server&name=BDTService&organization=ikurento.com&owner=ZX&pid=64225&retries=0&service.filter=echo&side=provider&timestamp=1562076628&version=&warmup=100")
	configurator.Configure(&providerUrl)
	assert.Equal(t, "failfast", providerUrl.Params.Get(constant.CLUSTER_KEY))

}
func Test_configureVerisonOverrideAddr(t *testing.T) {
	url, err := common.NewURL(context.Background(), "override://0.0.0.0:0/com.xxx.mock.userProvider?group=1&version=1&cluster=failfast&application=BDTService&providerAddresses=127.0.0.2:20001|127.0.0.3:20001")
	assert.NoError(t, err)
	configurator := extension.GetConfigurator("override", &url)
	assert.Equal(t, "override", configurator.GetUrl().Protocol)

	providerUrl, err := common.NewURL(context.Background(), "jsonrpc://127.0.0.1:20001/com.ikurento.user.UserProvider?anyhost=true&app.version=0.0.1&application=BDTService&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&group=&interface=com.ikurento.user.UserProvider&ip=10.32.20.124&loadbalance=random&methods.GetUser.loadbalance=random&methods.GetUser.retries=1&methods.GetUser.weight=0&module=dubbogo+user-info+server&name=BDTService&organization=ikurento.com&owner=ZX&pid=64225&retries=0&service.filter=echo&side=provider&timestamp=1562076628&version=&warmup=100")
	configurator.Configure(&providerUrl)
	assert.Equal(t, "failover", providerUrl.Params.Get(constant.CLUSTER_KEY))

}
func Test_configureVerison2p6WithIp(t *testing.T) {
	url, err := common.NewURL(context.Background(), "override://127.0.0.1:20001/com.xxx.mock.userProvider?group=1&version=1&cluster=failfast&application=BDTService")
	assert.NoError(t, err)
	configurator := extension.GetConfigurator("override", &url)
	assert.Equal(t, "override", configurator.GetUrl().Protocol)

	providerUrl, err := common.NewURL(context.Background(), "jsonrpc://127.0.0.1:20001/com.ikurento.user.UserProvider?anyhost=true&app.version=0.0.1&application=BDTService&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&group=&interface=com.ikurento.user.UserProvider&ip=10.32.20.124&loadbalance=random&methods.GetUser.loadbalance=random&methods.GetUser.retries=1&methods.GetUser.weight=0&module=dubbogo+user-info+server&name=BDTService&organization=ikurento.com&owner=ZX&pid=64225&retries=0&service.filter=echo&side=provider&timestamp=1562076628&version=&warmup=100")
	configurator.Configure(&providerUrl)
	assert.Equal(t, "failfast", providerUrl.Params.Get(constant.CLUSTER_KEY))

}

func Test_configureVerison2p7(t *testing.T) {
	url, err := common.NewURL(context.Background(), "jsonrpc://0.0.0.0:20001/com.xxx.mock.userProvider?group=1&version=1&cluster=failfast&application=BDTService&configVersion=1.0&side=provider")
	assert.NoError(t, err)
	configurator := extension.GetConfigurator("override", &url)

	providerUrl, err := common.NewURL(context.Background(), "jsonrpc://127.0.0.1:20001/com.ikurento.user.UserProvider?anyhost=true&app.version=0.0.1&application=BDTService&category=providers&cluster=failover&dubbo=dubbo-provider-golang-2.6.0&environment=dev&group=&interface=com.ikurento.user.UserProvider&ip=10.32.20.124&loadbalance=random&methods.GetUser.loadbalance=random&methods.GetUser.retries=1&methods.GetUser.weight=0&module=dubbogo+user-info+server&name=BDTService&organization=ikurento.com&owner=ZX&pid=64225&retries=0&service.filter=echo&side=provider&timestamp=1562076628&version=&warmup=100")
	configurator.Configure(&providerUrl)
	assert.Equal(t, "failfast", providerUrl.Params.Get(constant.CLUSTER_KEY))

}
