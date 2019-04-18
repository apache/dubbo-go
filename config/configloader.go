package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

import (
	"github.com/dubbo/dubbo-go/common/constant"
)

var (
	consumerConfig ConsumerConfig
	providerConfig ProviderConfig
)

// loaded comsumer & provider config from xxx.yml
// Namely: dubbo.comsumer.xml & dubbo.provider.xml
func init() {
	// consumer
	path := os.Getenv(constant.CONF_CONSUMER_FILE_PATH)
	if path == "" {
		return
	}

	file, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	con := &ConsumerConfig{}
	err = yaml.Unmarshal(file, con)
	if err != nil {
		panic(err)
	}

	consumerConfig = *con

	// provider
	path = os.Getenv(constant.CONF_PROVIDER_FILE_PATH)
	if path == "" {
		return
	}

	file, err = ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	pro := &ProviderConfig{}
	err = yaml.Unmarshal(file, pro)
	if err != nil {
		panic(err)
	}

	providerConfig = *pro
}

/////////////////////////
// consumerConfig
/////////////////////////

type ConsumerConfig struct {
	RegistryConfigs []map[string]string `yaml:"registryConfigs" json:"registryConfigs"`
}

func SetConsumerConfig(c ConsumerConfig) {
	consumerConfig = c
}
func GetConsumerConfig() ConsumerConfig {
	return consumerConfig
}

/////////////////////////
// providerConfig
/////////////////////////

type ProviderConfig struct {
	RegistryConfigs []map[string]string `yaml:"registryConfigs" json:"registryConfigs"`
}

func SetProviderConfig(p ProviderConfig) {
	providerConfig = p
}
func GetProviderConfig() ProviderConfig {
	return providerConfig
}
