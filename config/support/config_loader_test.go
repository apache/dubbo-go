package support

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
	assert.Nil(t, providerConfig)

	err = consumerInit(conPath)
	assert.NoError(t, err)
	err = providerInit(proPath)
	assert.NoError(t, err)

	assert.NotNil(t, consumerConfig)
	assert.NotNil(t, providerConfig)
}
