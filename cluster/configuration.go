package cluster

import "github.com/apache/dubbo-go/common"

type Configurator interface {
	GetUrl() *common.URL
	Configure(url *common.URL)
}
