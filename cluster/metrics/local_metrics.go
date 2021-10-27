package metrics

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"fmt"
)

type LocalMetrics struct {
	metrics map[string]interface{}
}

func (m *LocalMetrics) GetMethodMetrics(url *common.URL, methodName, key string) (interface{}, error) {
	mkey := fmt.Sprintf("%s.%s.%s.%s", getInstanceKey(url), getInvokerKey(url), methodName, key)
	panic(mkey)
}


func (m *LocalMetrics) GetInvokerMetrics(url *common.URL, key string) (interface{}, error) {
	panic("implement me")
}

func (m *LocalMetrics) GetInstanceMetrics(url *common.URL, key string) (interface{}, error) {
	panic("implement me")
}
