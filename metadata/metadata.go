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
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
)

var (
	exportOnce sync.Once
)

func ExportMetadataService(app, metadataType string) {
	exportOnce.Do(func() {
		err := service.Exporter.Export(app, metadataType)
		if err != nil {
			logger.Errorf("export metadata service failed, got error %#v", err)
		}
	})

	// err = publishMapping(expt)
	// if err != nil {
	// 	logger.Errorf("Publish interface-application mapping failed, got error %#v", err)
	// }
}

// OnEvent only handle ServiceConfigExportedEvent
// func publishMapping(sc exporter.MetadataServiceExporter) error {
// 	urls := sc.GetExportedURLs()

// 	for _, u := range urls {
// 		err := extension.GetGlobalServiceNameMapping().Map(u)
// 		if err != nil {
// 			return perrors.WithMessage(err, "could not map the service: "+u.String())
// 		}
// 	}
// 	return nil
// }
