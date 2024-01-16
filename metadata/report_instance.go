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
	"strings"
	"time"
)

import (
	"github.com/dubbogo/gost/container/set"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metadataMetrics "dubbo.apache.org/dubbo-go/v3/metrics/metadata"
)

var (
	instances = make(map[string]report.MetadataReport)
)

func toUrl(opts *ReportOptions) (*common.URL, error) {
	res, err := common.NewURL(opts.Address,
		common.WithUsername(opts.Username),
		common.WithPassword(opts.Password),
		common.WithLocation(opts.Address),
		common.WithProtocol(opts.Protocol),
		common.WithParamsValue(constant.TimeoutKey, opts.Timeout),
		common.WithParamsValue(constant.MetadataReportGroupKey, opts.Group),
		common.WithParamsValue(constant.MetadataReportNamespaceKey, opts.Namespace),
		common.WithParamsValue(constant.ClientNameKey, strings.Join([]string{constant.MetadataReportPrefix, opts.Protocol, opts.Address}, "-")),
	)
	if err != nil || len(res.Protocol) == 0 {
		return nil, perrors.New("Invalid MetadataReport Config.")
	}
	res.SetParam("metadata", res.Protocol)
	for key, val := range opts.Params {
		res.SetParam(key, val)
	}
	return res, nil
}

func GetMetadataReport() report.MetadataReport {
	for _, v := range instances {
		return v
	}
	return nil
}

func GetMetadataReportByRegistry(registry string) report.MetadataReport {
	if len(registry) == 0 {
		registry = constant.DefaultKey
	}
	if r, ok := instances[registry]; ok {
		return r
	}
	return GetMetadataReport()
}

func GetMetadataReports() []report.MetadataReport {
	reports := make([]report.MetadataReport, len(instances))
	index := 0
	for _, r := range instances {
		reports[index] = r
		index++
	}
	return reports
}

func GetMetadataType() string {
	return metadataOptions.metadataType
}

// DelegateMetadataReport is a absolute delegate for DelegateMetadataReport
type DelegateMetadataReport struct {
	instance report.MetadataReport
}

// PublishAppMetadata delegate publish metadata info
func (d *DelegateMetadataReport) PublishAppMetadata(application, revision string, meta *info.MetadataInfo) error {
	event := metadataMetrics.NewMetadataMetricTimeEvent(metadataMetrics.MetadataPush)
	err := d.instance.PublishAppMetadata(application, revision, meta)
	event.Succ = err == nil
	event.End = time.Now()
	metrics.Publish(event)
	return err
}

// GetAppMetadata delegate get metadata info
func (d *DelegateMetadataReport) GetAppMetadata(application, revision string) (*info.MetadataInfo, error) {
	event := metadataMetrics.NewMetadataMetricTimeEvent(metadataMetrics.MetadataSub)
	meta, err := d.instance.GetAppMetadata(application, revision)
	event.Succ = err == nil
	event.End = time.Now()
	metrics.Publish(event)
	return meta, err
}

func (d *DelegateMetadataReport) GetServiceAppMapping(application string, group string, listener mapping.MappingListener) (*gxset.HashSet, error) {
	return d.instance.GetServiceAppMapping(application, group, listener)
}

func (d *DelegateMetadataReport) RegisterServiceAppMapping(interfaceName, group string, application string) error {
	return d.instance.RegisterServiceAppMapping(interfaceName, group, application)
}

func (d *DelegateMetadataReport) RemoveServiceAppMappingListener(interfaceName, group string) error {
	return d.instance.RemoveServiceAppMappingListener(interfaceName, group)
}
