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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metadata/report/factory"
)

func TestGetMetadataReportInstanceByReg(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		reportInstance := GetMetadataReportByRegistryProtocol("mock")
		assert.Nil(t, reportInstance)
	})

	t.Run("normal", func(t *testing.T) {
		u, _ := common.NewURL("mock://127.0.0.1")
		extension.SetMetadataReportFactory("mock", func() factory.MetadataReportFactory {
			return &mockMetadataReportFactory{}
		})
		SetMetadataReportInstanceByReg(u)
		reportInstance := GetMetadataReportByRegistryProtocol("mock")
		assert.NotNil(t, reportInstance)
	})
}

func TestSetMetadataReportInstanceByReg(t *testing.T) {

	t.Run("exist", func(t *testing.T) {
		regInstances = make(map[string]report.MetadataReport, 8)
		u, _ := common.NewURL("mock://127.0.0.1")
		extension.SetMetadataReportFactory("mock", func() factory.MetadataReportFactory {
			return &mockMetadataReportFactory{}
		})
		SetMetadataReportInstanceByReg(u)
		SetMetadataReportInstanceByReg(u)
		assert.True(t, len(regInstances) == 1)
	})
}
