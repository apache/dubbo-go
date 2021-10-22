package metrics

import "dubbo.apache.org/dubbo-go/v3/protocol"

type LocalMetrics struct {
	metrics map[string]interface{}
}


func (m *LocalMetrics) GetMethodMetrics(ivk *protocol.Invoker, methodName, key string) (interface{}, error) {

}
