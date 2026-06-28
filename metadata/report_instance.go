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
	"sort"
	"sync"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metrics"
	metadataMetrics "dubbo.apache.org/dubbo-go/v3/metrics/metadata"
)

var (
	instances   = make(map[string]report.MetadataReport)
	instancesMu sync.RWMutex
)

// ClearMetadataReportInstances resets the package-level instances map.
// Intended for test isolation only; do not call in production code.
func ClearMetadataReportInstances() {
	instances = make(map[string]report.MetadataReport)
}

func addMetadataReport(registryId string, url *common.URL) error {
	fac := extension.GetMetadataReportFactory(url.Protocol)
	if fac == nil {
		logger.Warnf("[Metadata] no metadata report factory of protocol %s found, please check if the metadata report factory is imported", url.Protocol)
		return nil
	}
	mr := &DelegateMetadataReport{instance: fac.CreateMetadataReport(url)}
	instancesMu.Lock()
	instances[registryId] = mr
	instancesMu.Unlock()
	return nil
}

// GetMetadataReport returns a single metadata report for callers that lack
// registry context. It prefers the "default" registry's report; when absent
// it falls back to the lexicographically first registry id so the selection
// is always stable across calls.
func GetMetadataReport() report.MetadataReport {
	instancesMu.RLock()
	defer instancesMu.RUnlock()

	if r, ok := instances[constant.DefaultKey]; ok {
		return r
	}
	keys := make([]string, 0, len(instances))
	for k := range instances {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		return instances[keys[0]]
	}
	return nil
}

// GetMetadataReportByRegistry returns the metadata report bound to the given
// registry id. When the registry id is empty the caller has no registry context,
// so the stable default returned by GetMetadataReport is used. When a specific
// (non-empty) registry id is not found, it falls back to the "default" report
// if one exists. This handles the common case where a standalone metadata-report
// config is registered under "default" while named registries (e.g. nacos, zk)
// need to use it. nil is returned only when neither the specific id nor "default"
// is registered.
func GetMetadataReportByRegistry(registry string) report.MetadataReport {
	if len(registry) == 0 {
		return GetMetadataReport()
	}
	instancesMu.RLock()
	defer instancesMu.RUnlock()
	if r, ok := instances[registry]; ok {
		return r
	}
	if r, ok := instances[constant.DefaultKey]; ok {
		logger.Infof("[Metadata] no metadata report bound to registryId=%s, falling back to default", registry)
		return r
	}
	logger.Warnf("[Metadata] no metadata report found for registryId=%s", registry)
	return nil
}

func GetMetadataReports() []report.MetadataReport {
	instancesMu.RLock()
	defer instancesMu.RUnlock()
	reports := make([]report.MetadataReport, len(instances))
	index := 0
	for _, r := range instances {
		reports[index] = r
		index++
	}
	return reports
}

func GetMetadataType() string {
	if metadataOptions == nil || metadataOptions.metadataType == "" {
		return constant.DefaultMetadataStorageType
	}
	return metadataOptions.metadataType
}

// DelegateMetadataReport is a absolute delegate for DelegateMetadataReport
type DelegateMetadataReport struct {
	instance report.MetadataReport
}

// URL returns the URL of the underlying metadata report instance.
func (d *DelegateMetadataReport) URL() *common.URL {
	return d.instance.URL()
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

// UnPublishAppMetadata delegate unpublish metadata info
func (d *DelegateMetadataReport) UnPublishAppMetadata(application, revision string) error {
	return d.instance.UnPublishAppMetadata(application, revision)
}

// ListAppRevisions delegate list app revisions
func (d *DelegateMetadataReport) ListAppRevisions(application string) ([]report.AppRevision, error) {
	return d.instance.ListAppRevisions(application)
}
