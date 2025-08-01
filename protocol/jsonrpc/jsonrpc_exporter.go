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

package jsonrpc

import (
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
)

// JsonrpcExporter is JSON RPC exporter and  extends from base invoker.
type JsonrpcExporter struct {
	base.BaseExporter
}

// NewJsonrpcExporter creates JSON RPC exporter with @key, @invoker and @exporterMap
func NewJsonrpcExporter(key string, invoker base.Invoker, exporterMap *sync.Map) *JsonrpcExporter {
	return &JsonrpcExporter{
		BaseExporter: *base.NewBaseExporter(key, invoker, exporterMap),
	}
}

// UnExport exported JSON RPC service.
func (je *JsonrpcExporter) UnExport() {
	interfaceName := je.GetInvoker().GetURL().GetParam(constant.InterfaceKey, "")
	err := common.ServiceMap.UnRegister(interfaceName, JSONRPC, je.GetInvoker().GetURL().ServiceKey())
	if err != nil {
		logger.Errorf("[JsonrpcExporter.UnExport] error: %v", err)
	}
	je.BaseExporter.UnExport()
}
