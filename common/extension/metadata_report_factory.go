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

package extension

import (
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

var metaDataReportFactories = NewRegistry[func() report.MetadataReportFactory]("metadata report factory")

// SetMetadataReportFactory sets the MetadataReportFactory with @name
func SetMetadataReportFactory(name string, v func() report.MetadataReportFactory) {
	metaDataReportFactories.Register(name, v)
}

// GetMetadataReportFactory finds the MetadataReportFactory with @name
func GetMetadataReportFactory(name string) report.MetadataReportFactory {
	v, ok := metaDataReportFactories.Get(name)
	if !ok {
		return nil
	}
	return v()
}
