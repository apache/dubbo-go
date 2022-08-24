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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

var (
	instance  report.MetadataReport
	reportUrl *common.URL
	once      sync.Once
)

func GetMetadataReportInstance() report.MetadataReport {
	if instance != nil {
		return instance
	}

	return GetMetadataReportByRegistryProtocol("")
}

// SetMetadataReportInstance, init metadat report instance
func SetMetadataReportInstance(selectiveUrl ...*common.URL) {
	once.Do(func() {
		var url *common.URL
		if len(selectiveUrl) > 0 {
			url = selectiveUrl[0]
			fac := extension.GetMetadataReportFactory(url.Protocol)
			if fac != nil {
				instance = fac.CreateMetadataReport(url)
			}
			reportUrl = url
		}
	})
}

// GetMetadataReportUrl will return the report instance url
func GetMetadataReportUrl() *common.URL {
	return reportUrl
}

// SetMetadataReportUrl will only can be used by unit test to mock url
func SetMetadataReportUrl(url *common.URL) {
	reportUrl = url
}
