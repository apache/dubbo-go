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

package protocol

import (
	"sync"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
)

// Extension - protocol
type Protocol interface {
	Export(invoker Invoker) Exporter
	Refer(url common.URL) Invoker
	Destroy()
}

// wrapping invoker
type Exporter interface {
	GetInvoker() Invoker
	Unexport()
}

/////////////////////////////
// base protocol
/////////////////////////////

type BaseProtocol struct {
	exporterMap *sync.Map
	invokers    []Invoker
}

func NewBaseProtocol() BaseProtocol {
	return BaseProtocol{
		exporterMap: new(sync.Map),
	}
}

func (bp *BaseProtocol) SetExporterMap(key string, exporter Exporter) {
	bp.exporterMap.Store(key, exporter)
}

func (bp *BaseProtocol) ExporterMap() *sync.Map {
	return bp.exporterMap
}

func (bp *BaseProtocol) SetInvokers(invoker Invoker) {
	bp.invokers = append(bp.invokers, invoker)
}

func (bp *BaseProtocol) Invokers() []Invoker {
	return bp.invokers
}

func (bp *BaseProtocol) Export(invoker Invoker) Exporter {
	return NewBaseExporter("base", invoker, bp.exporterMap)
}

func (bp *BaseProtocol) Refer(url common.URL) Invoker {
	return NewBaseInvoker(url)
}

// Destroy will destroy all invoker and exporter, so it only is called once.
func (bp *BaseProtocol) Destroy() {
	// destroy invokers
	for _, invoker := range bp.invokers {
		if invoker != nil {
			invoker.Destroy()
		}
	}
	bp.invokers = []Invoker{}

	// unexport exporters
	bp.exporterMap.Range(func(key, exporter interface{}) bool {
		if exporter != nil {
			exporter.(Exporter).Unexport()
		} else {
			bp.exporterMap.Delete(key)
		}
		return true
	})
}

/////////////////////////////
// base exporter
/////////////////////////////

type BaseExporter struct {
	key         string
	invoker     Invoker
	exporterMap *sync.Map
}

func NewBaseExporter(key string, invoker Invoker, exporterMap *sync.Map) *BaseExporter {
	return &BaseExporter{
		key:         key,
		invoker:     invoker,
		exporterMap: exporterMap,
	}
}

func (de *BaseExporter) GetInvoker() Invoker {
	return de.invoker

}

func (de *BaseExporter) Unexport() {
	logger.Infof("Exporter unexport.")
	de.invoker.Destroy()
	de.exporterMap.Delete(de.key)
}
