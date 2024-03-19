package processor

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config/interfaces"
	"dubbo.apache.org/dubbo-go/v3/istio"
	"github.com/dubbogo/gost/log/logger"
	"sync"
)

var (
	once           sync.Once
	pilotProcessor interfaces.ConfigPostProcessor
)

type pilotAgentProcessor struct {
}

func init() {
	extension.SetConfigPostProcessor("pilotAgent", newPilotAgentProcessor())
}

func newPilotAgentProcessor() interfaces.ConfigPostProcessor {
	if pilotProcessor == nil {
		once.Do(func() {
			pilotProcessor = &pilotAgentProcessor{}
		})
	}
	return pilotProcessor
}

func (p *pilotAgentProcessor) PostProcessReferenceConfig(url *common.URL) {
	if url.GetParamBool(constant.XdsKey, false) {
		logger.Infof("[PostProcessReferenceConfig] init pilot agent...")
		_, err := istio.GetPilotAgent(istio.PilotAgentTypeClientWorkload)
		if err != nil {
			logger.Errorf("[PostProcessReferenceConfig] init pilot agent error:%v", err)
		}
	}
}

func (p *pilotAgentProcessor) PostProcessServiceConfig(url *common.URL) {
	if url.GetParamBool(constant.XdsKey, false) {
		logger.Infof("[PostProcessServiceConfig] init pilot agent...")
		_, err := istio.GetPilotAgent(istio.PilotAgentTypeServerWorkload)
		if err != nil {
			logger.Errorf("[PostProcessServiceConfig] init pilot agent error:%v", err)
		}
	}
}
