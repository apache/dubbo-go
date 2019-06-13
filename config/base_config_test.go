package config

import (
	"github.com/apache/dubbo-go/common/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_refresh(t *testing.T) {
	//Id         string `required:"true" yaml:"id"  json:"id,omitempty"`
	//Type       string `required:"true" yaml:"type"  json:"type,omitempty"`
	//TimeoutStr string `yaml:"timeout" default:"5s" json:"timeout,omitempty"` // unit: second
	//Group      string `yaml:"group" json:"group,omitempty"`
	////for registry
	//Address  string `yaml:"address" json:"address,omitempty"`
	//Username string `yaml:"username" json:"address,omitempty"`
	//Password string `yaml:"password" json:"address,omitempty"`

	c := &baseConfig{}
	mockMap := map[string]string{}
	mockMap["dubbo.registry.type"] = "zookeeper"
	mockMap["dubbo.registry.timeout"] = "3s"
	mockMap["dubbo.registry.group"] = "hangzhou"
	mockMap["dubbo.registry.address"] = "zookeeper://172.0.0.1:2181"
	mockMap["dubbo.registry.username"] = "admin"
	mockMap["dubbo.registry.password"] = "admin"
	config.GetEnvInstance().UpdateExternalConfigMap(mockMap)
	father := &RegistryConfig{Type: "1111"}
	c.SetFatherConfig(father)
	c.fresh()
	assert.Equal(t, "zookeeper", father.Type)
	assert.Equal(t, "zookeeper", father.Type)
}

func Test_refreshWithPrefix(t *testing.T) {
	//Id         string `required:"true" yaml:"id"  json:"id,omitempty"`
	//Type       string `required:"true" yaml:"type"  json:"type,omitempty"`
	//TimeoutStr string `yaml:"timeout" default:"5s" json:"timeout,omitempty"` // unit: second
	//Group      string `yaml:"group" json:"group,omitempty"`
	////for registry
	//Address  string `yaml:"address" json:"address,omitempty"`
	//Username string `yaml:"username" json:"address,omitempty"`
	//Password string `yaml:"password" json:"address,omitempty"`

	c := &baseConfig{}
	mockMap := map[string]string{}
	mockMap["dubbo.customRegistry.type"] = "zookeeper"
	mockMap["dubbo.customRegistry.timeout"] = "3s"
	mockMap["dubbo.customRegistry.group"] = "hangzhou"
	mockMap["dubbo.customRegistry.address"] = "zookeeper://172.0.0.1:2181"
	mockMap["dubbo.customRegistry.username"] = "admin"
	mockMap["dubbo.customRegistry.password"] = "admin"
	config.GetEnvInstance().UpdateExternalConfigMap(mockMap)
	father := &RegistryConfig{Type: "1111"}
	c.SetPrefix("dubbo.customRegistry")
	c.SetFatherConfig(father)
	c.fresh()
	assert.Equal(t, "zookeeper", father.Type)
	assert.Equal(t, "zookeeper", father.Type)
}
