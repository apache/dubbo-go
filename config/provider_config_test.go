package config

import (
	"path/filepath"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestProviderInit(t *testing.T) {
	conPath, err := filepath.Abs("./testdata/consumer_config_with_configcenter.yml")
	assert.NoError(t, err)
	assert.NoError(t, ConsumerInit(conPath))
	assert.Equal(t, "default", consumerConfig.ProxyFactory)
	assert.Equal(t, "dubbo.properties", consumerConfig.ConfigCenterConfig.ConfigFile)
	assert.Equal(t, "100ms", consumerConfig.Connect_Timeout)
}
