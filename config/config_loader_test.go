package config

import (
	"path/filepath"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestConfigLoader(t *testing.T) {
	conPath, err := filepath.Abs("./consumer_config.yml")
	assert.NoError(t, err)
	proPath, err := filepath.Abs("./provider_config.yml")
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
