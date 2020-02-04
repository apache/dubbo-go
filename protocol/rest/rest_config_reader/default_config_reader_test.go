package rest_config_reader

import (
	"os"
	"testing"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigReader_ReadConsumerConfig(t *testing.T) {
	err := os.Setenv(constant.CONF_CONSUMER_FILE_PATH, "./testdata/consumer_config.yml")
	assert.NoError(t, err)
	reader := GetDefaultConfigReader()
	config := reader.ReadConsumerConfig()
	assert.NotEmpty(t, config)
}

func TestDefaultConfigReader_ReadProviderConfig(t *testing.T) {
	err := os.Setenv(constant.CONF_PROVIDER_FILE_PATH, "./testdata/provider_config.yml")
	assert.NoError(t, err)
	reader := GetDefaultConfigReader()
	config := reader.ReadProviderConfig()
	assert.NotEmpty(t, config)
}
