package rest

import (
	"sync"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

type RestExporter struct {
	protocol.BaseExporter
}

func NewRestExporter(key string, invoker protocol.Invoker, exporterMap *sync.Map) *RestExporter {
	return &RestExporter{
		BaseExporter: *protocol.NewBaseExporter(key, invoker, exporterMap),
	}
}

func (re *RestExporter) Unexport() {
	serviceId := re.GetInvoker().GetUrl().GetParam(constant.BEAN_NAME_KEY, "")
	re.BaseExporter.Unexport()
	err := common.ServiceMap.UnRegister(REST, serviceId)
	if err != nil {
		logger.Errorf("[RestExporter.Unexport] error: %v", err)
	}
	return
}
