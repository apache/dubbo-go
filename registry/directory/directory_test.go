package directory

import (
	"context"
	"net/url"
	"strconv"
	"testing"
	"time"
)
import (
	"github.com/stretchr/testify/assert"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/cluster/support"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/config"
	"github.com/dubbo/go-for-apache-dubbo/protocol/protocolwrapper"
	"github.com/dubbo/go-for-apache-dubbo/registry"
)

func TestSubscribe(t *testing.T) {
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)

	url, _ := config.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl, _ := config.NewURL(context.TODO(), "dubbo://127.0.0.1:20000")
	url.SubURL = &suburl
	mockRegistry := registry.NewMockRegistry()
	registryDirectory, _ := NewRegistryDirectory(&url, mockRegistry)

	go registryDirectory.Subscribe(*config.NewURLWithOptions("testservice"))
	for i := 0; i < 3; i++ {
		mockRegistry.MockEvent(&registry.ServiceEvent{Action: registry.ServiceAdd, Service: *config.NewURLWithOptions("TEST"+strconv.FormatInt(int64(i), 10), config.WithProtocol("dubbo"))})
	}

	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 3)
}

func TestSubscribe_Delete(t *testing.T) {
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)

	url, _ := config.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl, _ := config.NewURL(context.TODO(), "dubbo://127.0.0.1:20000")
	url.SubURL = &suburl
	mockRegistry := registry.NewMockRegistry()
	registryDirectory, _ := NewRegistryDirectory(&url, mockRegistry)

	go registryDirectory.Subscribe(*config.NewURLWithOptions("testservice"))
	for i := 0; i < 3; i++ {
		mockRegistry.MockEvent(&registry.ServiceEvent{Action: registry.ServiceAdd, Service: *config.NewURLWithOptions("TEST"+strconv.FormatInt(int64(i), 10), config.WithProtocol("dubbo"))})
	}
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 3)
	mockRegistry.MockEvent(&registry.ServiceEvent{Action: registry.ServiceDel, Service: *config.NewURLWithOptions("TEST0", config.WithProtocol("dubbo"))})
	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 2)

}
func TestSubscribe_InvalidUrl(t *testing.T) {
	url, _ := config.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	mockRegistry := registry.NewMockRegistry()
	_, err := NewRegistryDirectory(&url, mockRegistry)
	assert.Error(t, err)
}

func TestSubscribe_Group(t *testing.T) {
	extension.SetProtocol(protocolwrapper.FILTER, protocolwrapper.NewMockProtocolFilter)
	extension.SetCluster("mock", cluster.NewMockCluster)

	regurl, _ := config.NewURL(context.TODO(), "mock://127.0.0.1:1111")
	suburl, _ := config.NewURL(context.TODO(), "dubbo://127.0.0.1:20000")
	suburl.Params.Set(constant.CLUSTER_KEY, "mock")
	regurl.SubURL = &suburl
	mockRegistry := registry.NewMockRegistry()
	registryDirectory, _ := NewRegistryDirectory(&regurl, mockRegistry)

	go registryDirectory.Subscribe(*config.NewURLWithOptions("testservice"))

	//for group1
	urlmap := url.Values{}
	urlmap.Set(constant.GROUP_KEY, "group1")
	urlmap.Set(constant.CLUSTER_KEY, "failover") //to test merge url
	for i := 0; i < 3; i++ {
		mockRegistry.MockEvent(&registry.ServiceEvent{Action: registry.ServiceAdd, Service: *config.NewURLWithOptions("TEST"+strconv.FormatInt(int64(i), 10), config.WithProtocol("dubbo"),
			config.WithParams(urlmap))})
	}
	//for group2
	urlmap2 := url.Values{}
	urlmap2.Set(constant.GROUP_KEY, "group2")
	urlmap2.Set(constant.CLUSTER_KEY, "failover") //to test merge url
	for i := 0; i < 3; i++ {
		mockRegistry.MockEvent(&registry.ServiceEvent{Action: registry.ServiceAdd, Service: *config.NewURLWithOptions("TEST"+strconv.FormatInt(int64(i), 10), config.WithProtocol("dubbo"),
			config.WithParams(urlmap2))})
	}

	time.Sleep(1e9)
	assert.Len(t, registryDirectory.cacheInvokers, 2)
}
