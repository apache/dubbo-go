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

package dubbo3

import (
	"sync"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

// DubboExporter is dubbo service exporter.
type Dubbo3Exporter struct {
	protocol.BaseExporter
}

// NewDubbo3Exporter get a DubboExporter.
func NewDubbo3Exporter(key string, invoker protocol.Invoker, exporterMap *sync.Map) *Dubbo3Exporter {
	return &Dubbo3Exporter{
		BaseExporter: *protocol.NewBaseExporter(key, invoker, exporterMap),
	}
}

// Unexport unexport dubbo service exporter.
func (de *Dubbo3Exporter) Unexport() {
	serviceId := de.GetInvoker().GetUrl().GetParam(constant.BEAN_NAME_KEY, "")
	interfaceName := de.GetInvoker().GetUrl().GetParam(constant.INTERFACE_KEY, "")
	de.BaseExporter.Unexport()
	err := common.ServiceMap.UnRegister(interfaceName, DUBBO3, serviceId)
	if err != nil {
		logger.Errorf("[DubboExporter.Unexport] error: %v", err)
	}
}
