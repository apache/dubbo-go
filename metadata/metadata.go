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

// Package metadata collects and exposes information of all services for service discovery purpose.
package metadata

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
)

var (
	metadataService    MetadataService = &DefaultMetadataService{}
	appMetadataInfoMap                 = make(map[string]*info.MetadataInfo)
)

func GetMetadataService() MetadataService {
	return metadataService
}

func GetMetadataInfo(registryId string) *info.MetadataInfo {
	return appMetadataInfoMap[registryId]
}

func AddService(registryId string, url *common.URL) {
	if _, exist := appMetadataInfoMap[registryId]; !exist {
		appMetadataInfoMap[registryId] = info.NewMetadataInfo(
			url.GetParam(constant.ApplicationKey, ""),
			url.GetParam(constant.ApplicationTagKey, ""),
		)
	}
	appMetadataInfoMap[registryId].AddService(url)
}

func AddSubscribeURL(registryId string, url *common.URL) {
	if _, exist := appMetadataInfoMap[registryId]; !exist {
		appMetadataInfoMap[registryId] = info.NewMetadataInfo(
			url.GetParam(constant.ApplicationKey, ""),
			url.GetParam(constant.ApplicationTagKey, ""),
		)
	}
	appMetadataInfoMap[registryId].AddSubscribeURL(url)
}
