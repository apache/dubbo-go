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

package metadata

import (
	"errors"
	"sync"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

const DefaultGroup = "mapping"

// retry policy for mapping registration. These are vars rather than consts so they can be
// tuned (and made near-instant in tests).
var (
	retryTimes        = 10
	retryBaseInterval = 100 * time.Millisecond
	retryMaxInterval  = 2 * time.Second
)

func init() {
	extension.SetGlobalServiceNameMapping(GetNameMappingInstance)
}

var (
	serviceNameMappingInstance *ServiceNameMapping
	serviceNameMappingOnce     sync.Once
)

// GetNameMappingInstance return an instance, if not found, it creates one
func GetNameMappingInstance() mapping.ServiceNameMapping {
	serviceNameMappingOnce.Do(func() {
		serviceNameMappingInstance = &ServiceNameMapping{}
	})
	return serviceNameMappingInstance
}

// ServiceNameMapping is the implementation based on metadata report
// it's a singleton
type ServiceNameMapping struct {
}

// Map will map the service to this application-level service
func (d *ServiceNameMapping) Map(url *common.URL) error {
	serviceInterface := url.GetParam(constant.InterfaceKey, "")
	appName := url.GetParam(constant.ApplicationKey, "")
	// url is the service url,not the registry url,this url has no registry id info,can not get where to write mapping,so write all
	// if the mapping can hold a report instance, it can write once
	metadataReports := metadata.GetMetadataReports()
	if len(metadataReports) == 0 {
		err := perrors.New("can not registering mapping to remote cause no metadata report instance found")
		logger.Errorf("[Metadata][Mapping] register failed interface=%s application=%s group=%s reports=0 err=%v", serviceInterface, appName, DefaultGroup, err)
		return err
	}
	for _, metadataReport := range metadataReports {
		if err := registerWithRetry(metadataReport, serviceInterface, DefaultGroup, appName); err != nil {
			logger.Errorf("[Metadata][Mapping] register failed interface=%s application=%s group=%s reports=%d err=%v", serviceInterface, appName, DefaultGroup, len(metadataReports), err)
			return err
		}
	}
	logger.Debugf("[Metadata][Mapping] register succeeded interface=%s application=%s group=%s reports=%d", serviceInterface, appName, DefaultGroup, len(metadataReports))
	return nil
}

// registerWithRetry registers the interface-to-app mapping, retrying only on CAS conflicts
// (report.ErrMappingCASConflict) with exponential backoff. Any other error is returned
// immediately, since retrying it would not help.
func registerWithRetry(r report.MetadataReport, serviceInterface, group, appName string) error {
	var err error
	for i := range retryTimes {
		err = r.RegisterServiceAppMapping(serviceInterface, group, appName)
		if err == nil {
			return nil
		}
		if !errors.Is(err, report.ErrMappingCASConflict) {
			return err
		}
		time.Sleep(backoff(i))
	}
	return err
}

// backoff returns the delay before retry attempt i: retryBaseInterval*2^i capped at
// retryMaxInterval.
func backoff(attempt int) time.Duration {
	d := retryBaseInterval << attempt
	if d <= 0 || d > retryMaxInterval {
		d = retryMaxInterval
	}
	return d
}

// Get will return the application-level services. If not found, the empty set will be returned.
func (d *ServiceNameMapping) Get(url *common.URL, listener mapping.MappingListener) (*gxset.HashSet, error) {
	serviceInterface := url.GetParam(constant.InterfaceKey, "")
	metadataReports := metadata.GetMetadataReports()
	if len(metadataReports) == 0 {
		err := perrors.New("can not get mapping in remote cause no metadata report instance found")
		logger.Warnf("[Metadata][Mapping] get failed interface=%s group=%s reports=0 err=%v", serviceInterface, DefaultGroup, err)
		return nil, err
	}
	operation := "get"
	if listener != nil {
		operation = "listen"
	}
	// Attach the listener to the stable primary report only (GetMetadataReport uses
	// a deterministic selection: prefer "default", otherwise lexicographic first).
	// GetMetadataReports() iterates a map so its order is non-deterministic; using
	// i==0 as the anchor would bind the listener to a random backend each run.
	primaryReport := metadata.GetMetadataReport()
	var result *gxset.HashSet
	var errs []error
	for _, metadataReport := range metadataReports {
		var reportListener mapping.MappingListener
		if metadataReport == primaryReport {
			reportListener = listener
		}
		set, err := metadataReport.GetServiceAppMapping(serviceInterface, DefaultGroup, reportListener)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if result == nil {
			result = set
		} else {
			result.Add(set.Values()...)
		}
	}
	if result == nil {
		if err := errors.Join(errs...); err != nil {
			logger.Warnf("[Metadata][Mapping] %s failed interface=%s group=%s reports=%d err=%v", operation, serviceInterface, DefaultGroup, len(metadataReports), err)
			return nil, err
		}
		return nil, nil
	}
	logger.Debugf("[Metadata][Mapping] %s succeeded interface=%s group=%s reports=%d apps=%d", operation, serviceInterface, DefaultGroup, len(metadataReports), result.Size())
	return result, nil
}

// Remove removes the service-to-app mapping for the given URL from all
// registered metadata reports. Unlike Map (which stops on the first failure),
// Remove is best-effort: it attempts every report and returns all errors
// joined together so the caller can see the full failure picture. The
// intent is to avoid leaving stale entries in any registry due to a transient
// error in one of the others.
func (d *ServiceNameMapping) Remove(url *common.URL) error {
	serviceInterface := url.GetParam(constant.InterfaceKey, "")
	metadataReports := metadata.GetMetadataReports()
	if len(metadataReports) == 0 {
		err := perrors.New("can not remove mapping in remote cause no metadata report instance found")
		logger.Warnf("[Metadata][Mapping] remove failed interface=%s group=%s reports=0 err=%v", serviceInterface, DefaultGroup, err)
		return err
	}
	var errs []error
	for _, metadataReport := range metadataReports {
		if err := metadataReport.RemoveServiceAppMappingListener(serviceInterface, DefaultGroup); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		logger.Warnf("[Metadata][Mapping] remove failed interface=%s group=%s reports=%d err=%v", serviceInterface, DefaultGroup, len(metadataReports), err)
		return err
	}
	logger.Debugf("[Metadata][Mapping] remove succeeded interface=%s group=%s reports=%d", serviceInterface, DefaultGroup, len(metadataReports))
	return nil
}
