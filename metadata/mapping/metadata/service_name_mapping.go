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
		return perrors.New("can not registering mapping to remote cause no metadata report instance found")
	}
	for _, metadataReport := range metadataReports {
		if err := registerWithRetry(metadataReport, serviceInterface, DefaultGroup, appName); err != nil {
			return err
		}
	}
	return nil
}

// registerWithRetry registers the interface-to-app mapping, retrying only on CAS conflicts
// (report.ErrMappingCASConflict) with exponential backoff. Any other error is returned
// immediately, since retrying it would not help.
func registerWithRetry(r report.MetadataReport, serviceInterface, group, appName string) error {
	var err error
	for i := 0; i < retryTimes; i++ {
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
	metadataReport := metadata.GetMetadataReport()
	if metadataReport == nil {
		return nil, perrors.New("can not get mapping in remote cause no metadata report instance found")
	}
	return metadataReport.GetServiceAppMapping(serviceInterface, DefaultGroup, listener)
}

func (d *ServiceNameMapping) Remove(url *common.URL) error {
	serviceInterface := url.GetParam(constant.InterfaceKey, "")
	metadataReport := metadata.GetMetadataReport()
	if metadataReport == nil {
		return perrors.New("can not remove mapping in remote cause no metadata report instance found")
	}
	return metadataReport.RemoveServiceAppMappingListener(serviceInterface, DefaultGroup)
}
