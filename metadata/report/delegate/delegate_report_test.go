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

package delegate

import (
	"fmt"
	"testing"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/stretchr/testify/assert"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config/instance"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
)

func TestMetadataReport_MetadataReportRetry(t *testing.T) {
	counter := atomic.NewInt64(1)

	retry, err := newMetadataReportRetry(1, 10, func() bool {
		counter.Add(1)
		return true
	})
	assert.NoError(t, err)
	retry.startRetryTask()
	<-time.After(2000 * time.Millisecond)
	retry.scheduler.Clear()
	assert.Equal(t, int64(3), counter.Load())
	logger.Info("over")
}

func TestMetadataReport_MetadataReportRetryWithLimit(t *testing.T) {
	counter := atomic.NewInt64(1)

	retry, err := newMetadataReportRetry(1, 1, func() bool {
		counter.Add(1)
		return true
	})
	assert.NoError(t, err)
	retry.startRetryTask()
	<-time.After(2000 * time.Millisecond)
	retry.scheduler.Clear()
	assert.Equal(t, int64(2), counter.Load())
	logger.Info("over")
}

func mockNewMetadataReport(t *testing.T) *MetadataReport {
	syncReportKey := "false"
	retryPeriodKey := "3"
	retryTimesKey := "100"
	cycleReportKey := "true"

	url, err := common.NewURL(fmt.Sprintf(
		"test://127.0.0.1:20000/?"+constant.SyncReportKey+"=%v&"+constant.RetryPeriodKey+"=%v&"+
			constant.RetryTimesKey+"=%v&"+constant.CycleReportKey+"=%v",
		syncReportKey, retryPeriodKey, retryTimesKey, cycleReportKey))
	assert.NoError(t, err)
	instance.SetMetadataReportUrl(url)
	mtr, err := NewMetadataReport()
	assert.NoError(t, err)
	assert.NotNil(t, mtr)
	return mtr
}

func TestMetadataReport_StoreProviderMetadata(t *testing.T) {
	mtr := mockNewMetadataReport(t)
	metadataId := &identifier.MetadataIdentifier{
		Application: "app",
		BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
			ServiceInterface: "com.ikurento.user.UserProvider",
			Version:          "0.0.1",
			Group:            "group1",
			Side:             "provider",
		},
	}

	mtr.StoreProviderMetadata(metadataId, getMockDefinition(metadataId, t))
}

func getMockDefinition(id *identifier.MetadataIdentifier, t *testing.T) *definition.ServiceDefinition {
	protocol := "dubbo"
	beanName := "UserProvider"
	url, err := common.NewURL(fmt.Sprintf(
		"%v://127.0.0.1:20000/com.ikurento.user.UserProvider1?anyhost=true&"+
			"application=BDTService&category=providers&default.timeout=10000&dubbo=dubbo-provider-golang-1.0.0&"+
			"environment=dev&interface=%v&ip=192.168.56.1&methods=GetUser&module=dubbogo+user-info+server&org=ikurento.com&"+
			"owner=ZX&pid=1447&revision=0.0.1&side=provider&timeout=3000&timestamp=1556509797245&group=%v&version=%v&bean.name=%v",
		protocol, id.ServiceInterface, id.Group, id.Version, beanName))
	assert.NoError(t, err)
	_, err = common.ServiceMap.Register(id.ServiceInterface, protocol, id.Group, id.Version, &definition.UserProvider{})
	assert.NoError(t, err)
	service := common.ServiceMap.GetServiceByServiceKey(url.Protocol, url.ServiceKey())
	return definition.BuildServiceDefinition(*service, url)
}
