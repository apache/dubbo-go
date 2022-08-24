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

package instance

import (
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

var (
	regInstances = make(map[string]report.MetadataReport, 8)
	mux          sync.RWMutex
)

// GetMetadataReportByRegistryProtocol obtain metadata report instance through registry config
func GetMetadataReportByRegistryProtocol(protocol string) report.MetadataReport {
	mux.RLock()
	defer mux.RUnlock()
	if protocol == "" {
		// return the first instance
		for _, regInstance := range regInstances {
			return regInstance
		}
	}
	// find the accurate instance
	regInstance, ok := regInstances[protocol]
	if !ok {
		return nil
	}
	return regInstance
}

// SetMetadataReportInstanceByReg set metadata reporting instances by registering urls
func SetMetadataReportInstanceByReg(url *common.URL) {
	mux.Lock()
	defer mux.Unlock()
	if _, ok := regInstances[url.Protocol]; ok {
		return
	}

	fac := extension.GetMetadataReportFactory(url.Protocol)
	if fac != nil {
		regInstances[url.Protocol] = fac.CreateMetadataReport(url)
	} else {
		logger.Infof("Metadata of type %v not registered.", url.Protocol)
	}
}
