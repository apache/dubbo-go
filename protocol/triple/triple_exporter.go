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

package triple

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/dubbogo/gost/log/logger"
	"sync"
)

// TripleExporter wraps BaseExporter
type TripleExporter struct {
	*protocol.BaseExporter
}

func NewTripleExporter(key string, invoker protocol.Invoker, exporterMap *sync.Map) *TripleExporter {
	return &TripleExporter{
		BaseExporter: protocol.NewBaseExporter(key, invoker, exporterMap),
	}
}

// UnExport and unregister Triple service from registry and memory.
func (te *TripleExporter) UnExport() {
	interfaceName := te.GetInvoker().GetURL().GetParam(constant.InterfaceKey, "")
	te.BaseExporter.UnExport()
	// todo: move UnRegister logic to a better place
	err := common.ServiceMap.UnRegister(interfaceName, TRIPLE, te.GetInvoker().GetURL().ServiceKey())
	if err != nil {
		logger.Errorf("[GrpcNewExporter.UnExport] error: %v", err)
	}
}
