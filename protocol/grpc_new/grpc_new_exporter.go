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

package grpc_new

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/dubbogo/gost/log/logger"
	"sync"
)

// GrpcNewExporter wraps BaseExporter
type GrpcNewExporter struct {
	*protocol.BaseExporter
}

func NewGrpcNewExporter(key string, invoker protocol.Invoker, exporterMap *sync.Map) *GrpcNewExporter {
	return &GrpcNewExporter{
		BaseExporter: protocol.NewBaseExporter(key, invoker, exporterMap),
	}
}

// UnExport and unregister GRPC_NEW service from registry and memory.
func (gne *GrpcNewExporter) UnExport() {
	interfaceName := gne.GetInvoker().GetURL().GetParam(constant.InterfaceKey, "")
	gne.BaseExporter.UnExport()
	err := common.ServiceMap.UnRegister(interfaceName, GRPC_NEW, gne.GetInvoker().GetURL().ServiceKey())
	if err != nil {
		logger.Errorf("[GrpcNewExporter.UnExport] error: %v", err)
	}
}
