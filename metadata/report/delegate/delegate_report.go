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
	"encoding/json"
	"runtime/debug"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/go-co-op/gocron"

	perrors "github.com/pkg/errors"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config/instance"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
)

const (
	defaultMetadataReportRetryTimes  int64 = 100  // max  times to retry
	defaultMetadataReportRetryPeriod int64 = 3    // cycle interval to retry, the unit is second
	defaultMetadataReportCycleReport bool  = true // cycle report or not
)

// metadataReportRetry is a scheduler for retrying task
type metadataReportRetry struct {
	retryPeriod  int64
	retryLimit   int64
	scheduler    *gocron.Scheduler
	job          *gocron.Job
	retryCounter *atomic.Int64
	// if no failed report, wait how many times to run retry task.
	retryTimesIfNonFail int64
}

// newMetadataReportRetry will create a scheduler for retry task
func newMetadataReportRetry(retryPeriod int64, retryLimit int64, retryFunc func() bool) (*metadataReportRetry, error) {
	s1 := gocron.NewScheduler(time.UTC)

	mrr := &metadataReportRetry{
		retryPeriod:         retryPeriod,
		retryLimit:          retryLimit,
		scheduler:           s1,
		retryCounter:        atomic.NewInt64(0),
		retryTimesIfNonFail: 600,
	}

	newJob, err := mrr.scheduler.Every(int(mrr.retryPeriod)).Seconds().Do(
		func() {
			mrr.retryCounter.Inc()
			logger.Infof("start to retry task for metadata report. retry times: %v", mrr.retryCounter.Load())
			if mrr.retryCounter.Load() > mrr.retryLimit {
				mrr.scheduler.Clear()
			} else if retryFunc() && mrr.retryCounter.Load() > mrr.retryTimesIfNonFail {
				mrr.scheduler.Clear() // may not interrupt the running job
			}
		})

	mrr.job = newJob
	return mrr, err
}

// startRetryTask will make scheduler with retry task run
func (mrr *metadataReportRetry) startRetryTask() {
	mrr.scheduler.StartAt(time.Now().Add(500 * time.Millisecond))
	mrr.scheduler.StartAsync()
}

// MetadataReport is a absolute delegate for MetadataReport
type MetadataReport struct {
	reportUrl           *common.URL
	syncReport          bool
	metadataReportRetry *metadataReportRetry

	failedReports     map[*identifier.MetadataIdentifier]interface{}
	failedReportsLock sync.RWMutex

	// allMetadataReports store all the metdadata reports records in memory
	allMetadataReports     map[*identifier.MetadataIdentifier]interface{}
	allMetadataReportsLock sync.RWMutex
}

// NewMetadataReport will create a MetadataReport with initiation
func NewMetadataReport() (*MetadataReport, error) {
	url := instance.GetMetadataReportUrl()
	if url == nil {
		logger.Warn("the metadataReport URL is not configured, you should configure it.")
		return nil, perrors.New("the metadataReport URL is not configured, you should configure it.")
	}
	bmr := &MetadataReport{
		reportUrl:          url,
		syncReport:         url.GetParamBool(constant.SyncReportKey, false),
		failedReports:      make(map[*identifier.MetadataIdentifier]interface{}, 4),
		allMetadataReports: make(map[*identifier.MetadataIdentifier]interface{}, 4),
	}

	mrr, err := newMetadataReportRetry(
		url.GetParamInt(constant.RetryPeriodKey, defaultMetadataReportRetryPeriod),
		url.GetParamInt(constant.RetryTimesKey, defaultMetadataReportRetryTimes),
		bmr.retry,
	)
	if err != nil {
		return nil, err
	}

	bmr.metadataReportRetry = mrr
	if url.GetParamBool(constant.CycleReportKey, defaultMetadataReportCycleReport) {
		scheduler := gocron.NewScheduler(time.UTC)
		_, err := scheduler.Every(1).Day().Do(
			func() {
				logger.Infof("start to publish all metadata in metadata report %s.", url.String())
				bmr.allMetadataReportsLock.RLock()
				bmr.doHandlerMetadataCollection(bmr.allMetadataReports)
				bmr.allMetadataReportsLock.RUnlock()
			})
		if err != nil {
			return nil, err
		}
		scheduler.StartAt(time.Now().Add(500 * time.Millisecond))
		scheduler.StartAsync()
	}
	return bmr, nil
}

// PublishAppMetadata delegate publish metadata info
func (mr *MetadataReport) PublishAppMetadata(identifier *identifier.SubscriberMetadataIdentifier, info *common.MetadataInfo) error {
	report := instance.GetMetadataReportInstance()
	return report.PublishAppMetadata(identifier, info)
}

// GetAppMetadata delegate get metadata info
func (mr *MetadataReport) GetAppMetadata(identifier *identifier.SubscriberMetadataIdentifier) (*common.MetadataInfo, error) {
	report := instance.GetMetadataReportInstance()
	return report.GetAppMetadata(identifier)
}

// retry will do metadata failed reports collection by call metadata report sdk
func (mr *MetadataReport) retry() bool {
	mr.failedReportsLock.RLock()
	defer mr.failedReportsLock.RUnlock()
	return mr.doHandlerMetadataCollection(mr.failedReports)
}

// StoreProviderMetadata will delegate to call remote metadata's sdk to store provider service definition
func (mr *MetadataReport) StoreProviderMetadata(identifier *identifier.MetadataIdentifier, definer definition.ServiceDefiner) {
	if mr.syncReport {
		mr.storeMetadataTask(common.PROVIDER, identifier, definer)
	}
	go mr.storeMetadataTask(common.PROVIDER, identifier, definer)
}

// storeMetadataTask will delegate to call remote metadata's sdk to store
func (mr *MetadataReport) storeMetadataTask(role int, identifier *identifier.MetadataIdentifier, definer interface{}) {
	logger.Infof("publish provider identifier and definition:  Identifier :%v ; definition: %v .", identifier, definer)
	mr.allMetadataReportsLock.Lock()
	mr.allMetadataReports[identifier] = definer
	mr.allMetadataReportsLock.Unlock()

	mr.failedReportsLock.Lock()
	delete(mr.failedReports, identifier)
	mr.failedReportsLock.Unlock()
	// data is store the json marshaled definition
	var (
		data []byte
		err  error
	)

	defer func() {
		if r := recover(); r != nil {
			mr.failedReportsLock.Lock()
			mr.failedReports[identifier] = definer
			mr.failedReportsLock.Unlock()
			mr.metadataReportRetry.startRetryTask()
			logger.Errorf("Failed to put provider metadata %v in %v, cause: %v\n%s\n",
				identifier, string(data), r, string(debug.Stack()))
		}
	}()

	data, err = json.Marshal(definer)
	if err != nil {
		logger.Errorf("storeProviderMetadataTask error in stage json.Marshal, msg is %+v", err)
		panic(err)
	}
	report := instance.GetMetadataReportInstance()
	if role == common.PROVIDER {
		err = report.StoreProviderMetadata(identifier, string(data))
	} else if role == common.CONSUMER {
		err = report.StoreConsumerMetadata(identifier, string(data))
	}

	if err != nil {
		logger.Errorf("storeProviderMetadataTask error in stage call  metadata report to StoreProviderMetadata, msg is %+v", err)
	}
}

// StoreConsumerMetadata will delegate to call remote metadata's sdk to store consumer side service definition
func (mr *MetadataReport) StoreConsumerMetadata(identifier *identifier.MetadataIdentifier, definer map[string]string) {
	if mr.syncReport {
		mr.storeMetadataTask(common.CONSUMER, identifier, definer)
	}
	go mr.storeMetadataTask(common.CONSUMER, identifier, definer)
}

// SaveServiceMetadata will delegate to call remote metadata's sdk to save service metadata
func (mr *MetadataReport) SaveServiceMetadata(identifier *identifier.ServiceMetadataIdentifier, url *common.URL) error {
	report := instance.GetMetadataReportInstance()
	if mr.syncReport {
		return report.SaveServiceMetadata(identifier, url)
	}
	go func() {
		if err := report.SaveServiceMetadata(identifier, url); err != nil {
			logger.Warnf("report.SaveServiceMetadata(identifier:%v, url:%v) = error:%v", identifier, url, err)
		}
	}()
	return nil
}

// RemoveServiceMetadata will delegate to call remote metadata's sdk to remove service metadata
func (mr *MetadataReport) RemoveServiceMetadata(identifier *identifier.ServiceMetadataIdentifier) error {
	report := instance.GetMetadataReportInstance()
	if mr.syncReport {
		return report.RemoveServiceMetadata(identifier)
	}
	go func() {
		if err := report.RemoveServiceMetadata(identifier); err != nil {
			logger.Warnf("report.RemoveServiceMetadata(identifier:%v) = error:%v", identifier, err)
		}
	}()
	return nil
}

// GetExportedURLs will delegate to call remote metadata's sdk to get exported urls
func (mr *MetadataReport) GetExportedURLs(identifier *identifier.ServiceMetadataIdentifier) ([]string, error) {
	report := instance.GetMetadataReportInstance()
	return report.GetExportedURLs(identifier)
}

// SaveSubscribedData will delegate to call remote metadata's sdk to save subscribed data
func (mr *MetadataReport) SaveSubscribedData(identifier *identifier.SubscriberMetadataIdentifier, urls []*common.URL) error {
	urlStrList := make([]string, 0, len(urls))
	for _, url := range urls {
		urlStrList = append(urlStrList, url.String())
	}
	bytes, err := json.Marshal(urlStrList)
	if err != nil {
		return perrors.WithMessage(err, "Could not convert the array to json")
	}

	report := instance.GetMetadataReportInstance()
	if mr.syncReport {
		return report.SaveSubscribedData(identifier, string(bytes))
	}
	go func() {
		if err := report.SaveSubscribedData(identifier, string(bytes)); err != nil {
			logger.Warnf("report.SaveSubscribedData(identifier:%v, string(bytes):%v) = error: %v",
				identifier, string(bytes), err)
		}
	}()
	return nil
}

// GetSubscribedURLs will delegate to call remote metadata's sdk to get subscribed urls
func (mr *MetadataReport) GetSubscribedURLs(identifier *identifier.SubscriberMetadataIdentifier) ([]string, error) {
	report := instance.GetMetadataReportInstance()
	return report.GetSubscribedURLs(identifier)
}

// GetServiceDefinition will delegate to call remote metadata's sdk to get service definitions
func (mr *MetadataReport) GetServiceDefinition(identifier *identifier.MetadataIdentifier) (string, error) {
	report := instance.GetMetadataReportInstance()
	return report.GetServiceDefinition(identifier)
}

// doHandlerMetadataCollection will store metadata to metadata support with given metadataMap
func (mr *MetadataReport) doHandlerMetadataCollection(metadataMap map[*identifier.MetadataIdentifier]interface{}) bool {
	if len(metadataMap) == 0 {
		return true
	}
	for e := range metadataMap {
		if common.RoleType(common.PROVIDER).Role() == e.Side {
			mr.StoreProviderMetadata(e, metadataMap[e].(*definition.ServiceDefinition))
		} else if common.RoleType(common.CONSUMER).Role() == e.Side {
			mr.StoreConsumerMetadata(e, metadataMap[e].(map[string]string))
		}
	}
	return false
}
