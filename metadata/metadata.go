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
	"github.com/dubbogo/gost/log/logger"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
)

var (
	exporting = &atomic.Bool{}
)

func ExportMetadataService() {
	var (
		err  error
		ms   service.MetadataService
		msV1 service.MetadataServiceV1
		msV2 service.MetadataServiceV2
	)

	ms, err = extension.GetLocalMetadataService(constant.DefaultKey)
	if err != nil {
		logger.Warn("could not init metadata service", err)
		return
	}
	msV1, err = extension.GetLocalMetadataServiceV1(constant.MetadataServiceV1)
	if err != nil {
		logger.Warn("could not init metadata service", err)
		return
	}
	msV2, err = extension.GetLocalMetadataServiceV2(constant.MetadataServiceV2)
	if err != nil {
		logger.Warn("could not init metadata service", err)
		return
	}

	if exporting.Load() {
		return
	}

	// In theory, we can use sync.Once
	// But sync.Once is not reentrant.
	// Now the invocation chain is createRegistry -> tryInitMetadataService -> metadataServiceExporter.export
	// -> createRegistry -> initMetadataService...
	// So using sync.Once will result in dead lock
	exporting.Store(true)

	expt := extension.GetMetadataServiceExporter(constant.DefaultKey, ms, msV1, msV2)
	if expt == nil {
		logger.Warnf("get metadata service exporter failed, pls check if you import _ \"dubbo.apache.org/dubbo-go/v3/metadata/service/exporter/configurable\"")
		return
	}

	err = expt.Export()
	if err != nil {
		logger.Errorf("could not export the metadata service, err = %s", err.Error())
		return
	}
}
