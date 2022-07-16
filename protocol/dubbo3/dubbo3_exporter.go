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
	"github.com/dubbogo/gost/log/logger"

	tripleConstant "github.com/dubbogo/triple/pkg/common/constant"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// DubboExporter is dubbo3 service exporter.
type DubboExporter struct {
	protocol.BaseExporter
	// serviceMap
	serviceMap *sync.Map
}

// NewDubboExporter get a Dubbo3Exporter.
func NewDubboExporter(key string, invoker protocol.Invoker, exporterMap *sync.Map, serviceMap *sync.Map) *DubboExporter {
	return &DubboExporter{
		BaseExporter: *protocol.NewBaseExporter(key, invoker, exporterMap),
		serviceMap:   serviceMap,
	}
}

// Unexport unexport dubbo3 service exporter.
func (de *DubboExporter) Unexport() {
	url := de.GetInvoker().GetURL()
	interfaceName := url.GetParam(constant.InterfaceKey, "")
	de.BaseExporter.Unexport()
	err := common.ServiceMap.UnRegister(interfaceName, tripleConstant.TRIPLE, url.ServiceKey())
	if err != nil {
		logger.Errorf("[DubboExporter.Unexport] error: %v", err)
	}
	de.serviceMap.Delete(interfaceName)
}
