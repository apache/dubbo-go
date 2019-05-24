// Copyright 2016-2019 Yincheng Fang
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsonrpc

import (
	"sync"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/logger"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type JsonrpcExporter struct {
	protocol.BaseExporter
}

func NewJsonrpcExporter(key string, invoker protocol.Invoker, exporterMap *sync.Map) *JsonrpcExporter {
	return &JsonrpcExporter{
		BaseExporter: *protocol.NewBaseExporter(key, invoker, exporterMap),
	}
}

func (je *JsonrpcExporter) Unexport() {
	service := je.GetInvoker().GetUrl().GetParam(constant.INTERFACE_KEY, "")
	je.BaseExporter.Unexport()
	err := common.ServiceMap.UnRegister(JSONRPC, service)
	if err != nil {
		logger.Errorf("[JsonrpcExporter.Unexport] error: %v", err)
	}
}
