package etcdv3

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/observer"
	"github.com/apache/dubbo-go/common/observer/dispatcher"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/registry"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testName = "test"

func setUp() {
	config.GetBaseConfig().ServiceDiscoveries[testName] = &config.ServiceDiscoveryConfig{
		Protocol:  "etcdv3",
		RemoteRef: testName,
	}

	config.GetBaseConfig().Remotes[testName] = &config.RemoteConfig{
		Address:    "localhost:2380",
		TimeoutStr: "1000s",
	}
}

func Test_newEtcdV3ServiceDiscovery(t *testing.T) {
	name := constant.ETCDV3_KEY
	_, err := newEtcdV3ServiceDiscovery(name)

	// warn: log configure file name is nil
	assert.NotNil(t, err)

	sdc := &config.ServiceDiscoveryConfig{
		Protocol:  "etcdv3",
		RemoteRef: "mock",
	}
	config.GetBaseConfig().ServiceDiscoveries[name] = sdc

	_, err = newEtcdV3ServiceDiscovery(name)

	// RemoteConfig not found
	assert.NotNil(t, err)

	config.GetBaseConfig().Remotes["mock"] = &config.RemoteConfig{
		Address:    "localhost:2380",
		TimeoutStr: "10s",
	}

	res, err := newEtcdV3ServiceDiscovery(name)
	assert.Nil(t, err)
	assert.NotNil(t, res)
}

func TestEtcdV3ServiceDiscovery_Destroy(t *testing.T) {
	setUp()
	serviceDiscovery, err := extension.GetServiceDiscovery(constant.ETCDV3_KEY, testName)

	assert.Nil(t, err)
	assert.NotNil(t, serviceDiscovery)

	err = serviceDiscovery.Destroy()
	assert.Nil(t, err)
	assert.NotNil(t, serviceDiscovery.(*etcdV3ServiceDiscovery).client)
}

func TestEtcdV3ServiceDiscovery_CRUD(t *testing.T) {
	setUp()
	extension.SetEventDispatcher("mock", func() observer.EventDispatcher {
		return &dispatcher.MockEventDispatcher{}
	})

	extension.SetAndInitGlobalDispatcher("mock")

	serviceName := "service-name"
	id := "id"
	host := "host"
	port := 123
	instance := &registry.DefaultServiceInstance{
		Id:          id,
		ServiceName: serviceName,
		Host:        host,
		Port:        port,
		Enable:      true,
		Healthy:     true,
		Metadata:    nil,
	}

	// clean data

	serviceDiscovry, _ := extension.GetServiceDiscovery(constant.ETCDV3_KEY, testName)

	// clean data for local test
	serviceDiscovry.Unregister(&registry.DefaultServiceInstance{
		Id:          id,
		ServiceName: serviceName,
		Host:        host,
		Port:        port,
	})

	err := serviceDiscovry.Register(instance)
	assert.Nil(t, err)

	page := serviceDiscovry.GetHealthyInstancesByPage(serviceName, 0, 10, true)
	assert.NotNil(t, page)

	assert.Equal(t, 0, page.GetOffset())
	assert.Equal(t, 10, page.GetPageSize())
	assert.Equal(t, 1, page.GetDataSize())

	instance = page.GetData()[0].(*registry.DefaultServiceInstance)
	assert.NotNil(t, instance)
	assert.Equal(t, id, instance.GetId())
	assert.Equal(t, host, instance.GetHost())
	assert.Equal(t, port, instance.GetPort())
	assert.Equal(t, serviceName, instance.GetServiceName())
	assert.Equal(t, 0, len(instance.GetMetadata()))

	instance.Metadata["a"] = "b"

	err = serviceDiscovry.Update(instance)
	assert.Nil(t, err)

	pageMap := serviceDiscovry.GetRequestInstances([]string{serviceName}, 0, 1)
	assert.Equal(t, 1, len(pageMap))
	page = pageMap[serviceName]
	assert.NotNil(t, page)
	assert.Equal(t, 1, len(page.GetData()))

	instance = page.GetData()[0].(*registry.DefaultServiceInstance)
	v, _ := instance.Metadata["a"]
	assert.Equal(t, "b", v)

	// test dispatcher event
	err = serviceDiscovry.DispatchEventByServiceName(serviceName)
	assert.Nil(t, err)

	// test AddListener
	err = serviceDiscovry.AddListener(&registry.ServiceInstancesChangedListener{ServiceName: serviceName})
	assert.Nil(t, err)
}

func TestEtcdV3ServiceDiscovery_GetDefaultPageSize(t *testing.T) {
	setUp()
	serviceDiscovry, _ := extension.GetServiceDiscovery(constant.ETCDV3_KEY, testName)
	assert.Equal(t, registry.DefaultPageSize, serviceDiscovry.GetDefaultPageSize())
}
