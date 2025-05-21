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
	"github.com/dubbogo/gost/log/logger"
)

// Exporter is the interface that wraps the basic GetInvoker method and Destroy UnExport.
//
// GetInvoker method is to get invoker.
//
// UnExport is to un export an exported service
type Exporter interface {
	GetInvoker() Invoker
	UnExport()
}

// BaseExporter is default exporter implement.
type BaseExporter struct {
	key         string
	invoker     Invoker
	exporterMap *sync.Map
}

// NewBaseExporter creates a new BaseExporter
func NewBaseExporter(key string, invoker Invoker, exporterMap *sync.Map) *BaseExporter {
	return &BaseExporter{
		key:         key,
		invoker:     invoker,
		exporterMap: exporterMap,
	}
}

// GetInvoker gets invoker
func (de *BaseExporter) GetInvoker() Invoker {
	return de.invoker
}

// UnExport un export service.
func (de *BaseExporter) UnExport() {
	logger.Infof("Exporter unexport.")
	de.invoker.Destroy()
	de.exporterMap.Delete(de.key)
}
