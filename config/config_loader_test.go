package config

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"path/filepath"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster/cluster_impl"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
)

func TestConfigLoader(t *testing.T) {
	conPath, err := filepath.Abs("./testdata/consumer_config.yml")
	assert.NoError(t, err)
	proPath, err := filepath.Abs("./testdata/provider_config.yml")
	assert.NoError(t, err)

	assert.Nil(t, consumerConfig)
	assert.Equal(t, ConsumerConfig{}, GetConsumerConfig())
	assert.Nil(t, providerConfig)
	assert.Equal(t, ProviderConfig{}, GetProviderConfig())

	err = consumerInit(conPath)
	assert.NoError(t, err)
	err = providerInit(proPath)
	assert.NoError(t, err)

	assert.NotNil(t, consumerConfig)
	assert.NotEqual(t, ConsumerConfig{}, GetConsumerConfig())
	assert.NotNil(t, providerConfig)
	assert.NotEqual(t, ProviderConfig{}, GetProviderConfig())
}

func TestLoad(t *testing.T) {
	doInit()
	doinit()

	SetConService(&MockService{})
	SetProService(&MockService{})

	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster("registryAware", cluster_impl.NewRegistryAwareCluster)
	consumerConfig.References[0].Registries = []ConfigRegistry{"shanghai_reg1"}

	refConfigs, svcConfigs := Load()
	assert.NotEqual(t, 0, len(refConfigs))
	assert.NotEqual(t, 0, len(svcConfigs))

	conServices = map[string]common.RPCService{}
	proServices = map[string]common.RPCService{}
	common.ServiceMap.UnRegister("mock", "MockService")
	consumerConfig = nil
	providerConfig = nil
}
