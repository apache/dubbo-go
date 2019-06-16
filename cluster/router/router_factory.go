package router

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
)

func init() {
	extension.SetRouterFactory("conditionRouterFactory", NewConditionRouterFactory)
}

type ConditionRouterFactory struct {
}

func NewConditionRouterFactory() cluster.RouterFactory {
	return ConditionRouterFactory{}
}
func (c ConditionRouterFactory) GetRouter(url common.URL) (cluster.Router, error) {
	return newConditionRouter(url)
}
