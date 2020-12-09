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
	"github.com/go-co-op/gocron"
	perrors "github.com/pkg/errors"
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config/instance"
	"github.com/apache/dubbo-go/metadata/definition"
	"github.com/apache/dubbo-go/metadata/identifier"
)

const (
	// defaultMetadataReportRetryTimes is defined for max  times to retry
	defaultMetadataReportRetryTimes int64 = 100
	// defaultMetadataReportRetryPeriod is defined for cycle interval to retry, the unit is second
	defaultMetadataReportRetryPeriod int64 = 3
	// defaultMetadataReportRetryPeriod is defined for cycle report or not
	defaultMetadataReportCycleReport bool = true
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

	newJob, err := mrr.scheduler.Every(uint64(mrr.retryPeriod)).Seconds().Do(
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
	mrr.scheduler.Start()
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
	bmr := &MetadataReport{
		reportUrl:          url,
		syncReport:         url.GetParamBool(constant.SYNC_REPORT_KEY, false),
		failedReports:      make(map[*identifier.MetadataIdentifier]interface{}, 4),
		allMetadataReports: make(map[*identifier.MetadataIdentifier]interface{}, 4),
	}

	mrr, err := newMetadataReportRetry(
		url.GetParamInt(constant.RETRY_PERIOD_KEY, defaultMetadataReportRetryPeriod),
		url.GetParamInt(constant.RETRY_TIMES_KEY, defaultMetadataReportRetryTimes),
		bmr.retry,
	)

	if err != nil {
		return nil, err
	}

	bmr.metadataReportRetry = mrr
	if url.GetParamBool(constant.CYCLE_REPORT_KEY, defaultMetadataReportCycleReport) {
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
		scheduler.Start()
	}
	return bmr, nil
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
	logger.Infof("store provider metadata. Identifier :%v ; definition: %v .", identifier, definer)
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
		panic(err)
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
	go report.SaveServiceMetadata(identifier, url)
	return nil
}

// RemoveServiceMetadata will delegate to call remote metadata's sdk to remove service metadata
func (mr *MetadataReport) RemoveServiceMetadata(identifier *identifier.ServiceMetadataIdentifier) error {
	report := instance.GetMetadataReportInstance()
	if mr.syncReport {
		return report.RemoveServiceMetadata(identifier)
	}
	go report.RemoveServiceMetadata(identifier)
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
	go report.SaveSubscribedData(identifier, string(bytes))
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
