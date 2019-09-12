package config

import (
	"path/filepath"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestConsumerInit(t *testing.T) {
	conPath, err := filepath.Abs("./testdata/consumer_config_with_configcenter.yml")
	assert.NoError(t, err)
	assert.NoError(t, ConsumerInit(conPath))
	assert.Equal(t, "default", consumerConfig.ProxyFactory)
	assert.Equal(t, "dubbo.properties", consumerConfig.ConfigCenterConfig.ConfigFile)
	assert.Equal(t, "100ms", consumerConfig.Connect_Timeout)
}

func TestConsumerInitWithDefaultProtocol(t *testing.T) {
	conPath, err := filepath.Abs("./testdata/consumer_config_withoutProtocol.yml")
	assert.NoError(t, err)
	assert.NoError(t, ConsumerInit(conPath))
	assert.Equal(t, "dubbo", consumerConfig.References["UserProvider"].Protocol)
}

func TestProviderInitWithDefaultProtocol(t *testing.T) {
	conPath, err := filepath.Abs("./testdata/provider_config_withoutProtocol.yml")
	assert.NoError(t, err)
	assert.NoError(t, ProviderInit(conPath))
	assert.Equal(t, "dubbo", providerConfig.Services["UserProvider"].Protocol)
}
