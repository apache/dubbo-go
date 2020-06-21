package etcdv3

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/registry"
)

var testName = "test"

func setUp() {
	config.GetBaseConfig().ServiceDiscoveries[testName] = &config.ServiceDiscoveryConfig{
		Protocol:  "etcdv3",
		RemoteRef: testName,
	}

	config.GetBaseConfig().Remotes[testName] = &config.RemoteConfig{
		Address:    "localhost:2379",
		TimeoutStr: "10s",
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
		Address:    "localhost:2379",
		TimeoutStr: "10s",
	}

	res, err := newEtcdV3ServiceDiscovery(name)
	assert.Nil(t, err)
	assert.NotNil(t, res)
}

func TestEtcdV3ServiceDiscovery_GetDefaultPageSize(t *testing.T) {
	setUp()
	serviceDiscovry := &etcdV3ServiceDiscovery{}
	assert.Equal(t, registry.DefaultPageSize, serviceDiscovry.GetDefaultPageSize())
}
