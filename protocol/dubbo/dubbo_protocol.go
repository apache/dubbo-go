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
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/common/logger"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

const DUBBO = "dubbo"

func init() {
	extension.SetProtocol(DUBBO, GetProtocol)
}

var dubboProtocol *DubboProtocol

type DubboProtocol struct {
	protocol.BaseProtocol
	serverMap map[string]*Server
}

func NewDubboProtocol() *DubboProtocol {
	return &DubboProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*Server),
	}
}

func (dp *DubboProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetUrl()
	serviceKey := url.Key()
	exporter := NewDubboExporter(serviceKey, invoker, dp.ExporterMap())
	dp.SetExporterMap(serviceKey, exporter)
	logger.Infof("Export service: %s", url.String())

	// start server
	dp.openServer(url)
	return exporter
}

func (dp *DubboProtocol) Refer(url common.URL) protocol.Invoker {
	invoker := NewDubboInvoker(url, NewClient())
	dp.SetInvokers(invoker)
	logger.Infof("Refer service: %s", url.String())
	return invoker
}

func (dp *DubboProtocol) Destroy() {
	logger.Infof("DubboProtocol destroy.")

	dp.BaseProtocol.Destroy()

	// stop server
	for key, server := range dp.serverMap {
		delete(dp.serverMap, key)
		server.Stop()
	}
}

func (dp *DubboProtocol) openServer(url common.URL) {
	exporter, ok := dp.ExporterMap().Load(url.Key())
	if !ok {
		panic("[DubboProtocol]" + url.Key() + "is not existing")
	}
	srv := NewServer(exporter.(protocol.Exporter))
	dp.serverMap[url.Location] = srv
	srv.Start(url)
}

func GetProtocol() protocol.Protocol {
	if dubboProtocol != nil {
		return dubboProtocol
	}
	return NewDubboProtocol()
}
