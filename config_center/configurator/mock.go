package configurator

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/config_center"
)

func NewMockConfigurator(url *common.URL) config_center.Configurator {
	return &mockConfigurator{configuratorUrl: url}
}

type mockConfigurator struct {
	configuratorUrl *common.URL
}

func (c *mockConfigurator) GetUrl() *common.URL {
	return c.configuratorUrl
}

func (c *mockConfigurator) Configure(url *common.URL) {

}
