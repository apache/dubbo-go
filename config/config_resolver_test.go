package config

import (
	"testing"
)

import (
	"github.com/knadh/koanf"

	"github.com/stretchr/testify/assert"
)

func TestResolvePlaceHolder(t *testing.T) {
	t.Run("test resolver", func(t *testing.T) {
		conf := NewLoaderConf(WithPath("/Users/zlb/GolandProjects/dubbo-go/config/testdata/config/resolver/application.yaml"))
		koan := GetConfigResolver(conf)
		assert.Equal(t, koan.Get("dubbo.config-center.address"), koan.Get("dubbo.registries.nacos.address"))
		assert.Equal(t, koan.Get("localhost"), koan.Get("dubbo.protocols.dubbo.ip"))
		assert.Equal(t, nil, koan.Get("dubbo.registries.nacos.group"))

		rc := NewRootConfigBuilder().Build()
		err := koan.UnmarshalWithConf(rc.Prefix(), rc, koanf.UnmarshalConf{Tag: "yaml"})
		assert.Nil(t, err)
		assert.Equal(t, rc.ConfigCenter.Address, rc.Registries["nacos"].Address)
		//not exist, default
		assert.Equal(t, "", rc.Registries["nacos"].Group)

	})
}
