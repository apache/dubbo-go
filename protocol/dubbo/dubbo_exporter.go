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

package dubbo

import (
	"sync"
)

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type DubboExporter struct {
	protocol.BaseExporter
}

func NewDubboExporter(key string, invoker protocol.Invoker, exporterMap *sync.Map) *DubboExporter {
	return &DubboExporter{
		BaseExporter: *protocol.NewBaseExporter(key, invoker, exporterMap),
	}
}

func (de *DubboExporter) Unexport() {
	service := de.GetInvoker().GetUrl().GetParam(constant.INTERFACE_KEY, "")
	de.BaseExporter.Unexport()
	err := common.ServiceMap.UnRegister(DUBBO, service)
	if err != nil {
		log.Error("[DubboExporter.Unexport] error: %v", err)
	}
}
