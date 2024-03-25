/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package processor

import (
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config/interfaces"
	"dubbo.apache.org/dubbo-go/v3/istio"
	"github.com/dubbogo/gost/log/logger"
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
