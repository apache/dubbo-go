package config

import (
	"github.com/knadh/koanf"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestProfilesConfig_Prefix(t *testing.T) {
	profiles := &ProfilesConfig{}
	assert.Equal(t, profiles.Prefix(), constant.ProfilesConfigPrefix)
}

func TestLoaderConf_MergeConfig(t *testing.T) {
	rc := NewRootConfigBuilder().Build()
	conf := NewLoaderConf(WithPath("./testdata/config/active/application.yaml"))
	koan := GetConfigResolver(conf)
	koan = conf.MergeConfig(koan, nil)

	err := koan.UnmarshalWithConf(rc.Prefix(), rc, koanf.UnmarshalConf{Tag: "yaml"})
	assert.Nil(t, err)

	registries := rc.Registries
	assert.NotNil(t, registries)
	assert.Equal(t, registries["nacos"].Timeout, "10s")
	assert.Equal(t, registries["nacos"].Address, "nacos://127.0.0.1:8848")

	protocols := rc.Protocols
	assert.NotNil(t, protocols)
	assert.Equal(t, protocols["dubbo"].Name, "dubbo")
	assert.Equal(t, protocols["dubbo"].Port, "20000")

	consumer := rc.Consumer
	assert.NotNil(t, consumer)
	assert.Equal(t, consumer.References["helloService"].Protocol, "dubbo")
	assert.Equal(t, consumer.References["helloService"].InterfaceName, "org.github.dubbo.HelloService")
}
