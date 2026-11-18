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

// TripleExporter wraps BaseExporter
type TripleExporter struct {
	*base.BaseExporter
	once          sync.Once
	retire        func() error
	url           *common.URL
	key           string
	exporterMap   *sync.Map
	publicationMu *sync.Mutex
}

func NewTripleExporter(key string, invoker base.Invoker, exporterMap *sync.Map) *TripleExporter {
	return &TripleExporter{
		BaseExporter: base.NewBaseExporter(key, invoker, exporterMap),
		url:          invoker.GetURL().Clone(), key: key, exporterMap: exporterMap,
	}
}

// UnExport and unregister Triple service from registry and memory.
func (te *TripleExporter) UnExport() {
	te.once.Do(func() {
		if te.retire != nil {
			if err := te.retire(); err != nil {
				logger.Warnf("[Triple][Exporter] close serialization resources: %v", err)
			}
		}
		// Draining above must not hold the publication lock: an active call
		// may need to export another service before it can finish.
		func() {
			if te.publicationMu != nil {
				te.publicationMu.Lock()
				defer te.publicationMu.Unlock()
			}
			// An old export must not unregister or remove a newer replacement.
			if current, ok := te.exporterMap.Load(te.key); ok && current == te {
				interfaceName := te.url.GetParam(constant.InterfaceKey, "")
				if err := common.ServiceMap.UnRegister(interfaceName, TRIPLE, te.url.ServiceKey()); err != nil {
					logger.Errorf("[Triple][Exporter] unexport failed, err=%v", err)
				}
			}
			te.exporterMap.CompareAndDelete(te.key, te)
		}()
		te.GetInvoker().Destroy()
	})
}
