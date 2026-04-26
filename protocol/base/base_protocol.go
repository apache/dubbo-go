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

package base

import (
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// Protocol is the interface that wraps the basic Export, Refer and Destroy method.
type Protocol interface {
	// Export method is to export service for remote invocation
	Export(invoker Invoker) Exporter
	// Refer method is to refer a remote service
	Refer(url *common.URL) Invoker
	// Destroy method will destroy all invokers and exporters, so it only is called once.
	Destroy()
}

// RegistryUnregisterer is an optional protocol capability used during graceful shutdown.
// Implementations should only unregister exported services from the registry and must not
// destroy protocol servers or unexport providers as part of this step.
type RegistryUnregisterer interface {
	UnregisterRegistries()
}

// BaseProtocol is default protocol implement.
type BaseProtocol struct {
	exporterMap  *sync.Map
	invokersLock sync.RWMutex
	// invokers is protected by invokersLock; never expose the backing slice to callers.
	invokers []Invoker
}

// NewBaseProtocol creates a new BaseProtocol
func NewBaseProtocol() BaseProtocol {
	return BaseProtocol{
		exporterMap: new(sync.Map),
	}
}

// SetExporterMap set @exporter with @key to local memory.
func (bp *BaseProtocol) SetExporterMap(key string, exporter Exporter) {
	bp.exporterMap.Store(key, exporter)
}

// ExporterMap gets exporter map.
func (bp *BaseProtocol) ExporterMap() *sync.Map {
	return bp.exporterMap
}

// SetInvokers sets invoker into local memory
func (bp *BaseProtocol) SetInvokers(invoker Invoker) {
	bp.invokersLock.Lock()
	defer bp.invokersLock.Unlock()

	bp.invokers = append(bp.invokers, invoker)
}

// Invokers gets a snapshot of all invokers.
func (bp *BaseProtocol) Invokers() []Invoker {
	bp.invokersLock.RLock()
	defer bp.invokersLock.RUnlock()

	invokers := make([]Invoker, len(bp.invokers))
	copy(invokers, bp.invokers)
	return invokers
}

// Export is default export implement.
func (bp *BaseProtocol) Export(invoker Invoker) Exporter {
	return NewBaseExporter("base", invoker, bp.exporterMap)
}

// Refer is default refer implement.
func (bp *BaseProtocol) Refer(url *common.URL) Invoker {
	return NewBaseInvoker(url)
}

// Destroy will destroy all invoker and exporter, so it only is called once.
func (bp *BaseProtocol) Destroy() {
	// Do not hold invokersLock while calling Invoker.Destroy(); implementations may run protocol callbacks.
	bp.invokersLock.Lock()
	invokers := make([]Invoker, len(bp.invokers))
	copy(invokers, bp.invokers)
	bp.invokers = []Invoker{}
	bp.invokersLock.Unlock()

	// destroy invokers
	for _, invoker := range invokers {
		if invoker != nil {
			invoker.Destroy()
		}
	}

	// un export exporters
	bp.exporterMap.Range(func(key, exporter any) bool {
		if exporter != nil {
			exporter.(Exporter).UnExport()
		} else {
			bp.exporterMap.Delete(key)
		}
		return true
	})
}
