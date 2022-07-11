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

package mq

import (
	"sync"
)

import (
	tripleConstant "github.com/dubbogo/triple/pkg/common/constant"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// RocketMQExporter is rocketmq service exporter
type RocketMQExporter struct {
	protocol.BaseExporter
	serviceMap *sync.Map
}

func NewRocketMQExporter(key string, invoker protocol.Invoker, exporterMap *sync.Map, serviceMap *sync.Map) *RocketMQExporter {
	return &RocketMQExporter{
		BaseExporter: *protocol.NewBaseExporter(key, invoker, exporterMap),
		serviceMap:   serviceMap,
	}
}

func (exporter *RocketMQExporter) Unexport() {
	url := exporter.GetInvoker().GetURL()
	interfaceName := url.GetParam(constant.InterfaceKey, "")
	exporter.BaseExporter.Unexport()

	err := common.ServiceMap.UnRegister(interfaceName, tripleConstant.TRIPLE, url.ServiceKey())
	if err != nil {
		logger.Errorf("[RocketMQExporter.Unexport] error: %v", err)
	}
	exporter.serviceMap.Delete(interfaceName)
}
