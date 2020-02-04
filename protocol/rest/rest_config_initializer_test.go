package rest

import (
	"os"
	"testing"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/stretchr/testify/assert"
)

func TestGetRestConsumerServiceConfig(t *testing.T) {
	err := os.Setenv(constant.CONF_CONSUMER_FILE_PATH, "./rest_config_reader/testdata/consumer_config.yml")
	assert.NoError(t, err)
	initConsumerRestConfig()
	serviceConfig := GetRestConsumerServiceConfig("com.ikurento.user.UserProvider")
	assert.NotEmpty(t, serviceConfig)
}

func TestGetRestProviderServiceConfig(t *testing.T) {
	err := os.Setenv(constant.CONF_PROVIDER_FILE_PATH, "./rest_config_reader/testdata/provider_config.yml")
	assert.NoError(t, err)
	initProviderRestConfig()
	serviceConfig := GetRestProviderServiceConfig("com.ikurento.user.UserProvider")
	assert.NotEmpty(t, serviceConfig)
}
