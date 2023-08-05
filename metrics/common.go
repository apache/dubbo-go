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

package metrics

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

type MetricKey struct {
	Name string
	Desc string
}

func NewMetricKey(name string, desc string) *MetricKey {
	return &MetricKey{Name: name, Desc: desc}
}

type MetricLevel interface {
	Tags() map[string]string
}

type ApplicationMetricLevel struct {
	ApplicationName string
	Version         string
	GitCommitId     string
	Ip              string
	HostName        string
}

var applicationName string
var applicationVersion string

// cannot import rootConfig,may cause cycle import,so be it
func InitAppInfo(appName string, appVersion string) {
	applicationName = appName
	applicationVersion = appVersion
}

func GetApplicationLevel() *ApplicationMetricLevel {
	return &ApplicationMetricLevel{
		ApplicationName: applicationName,
		Version:         applicationVersion,
		Ip:              common.GetLocalIp(),
		HostName:        common.GetLocalHostName(),
		GitCommitId:     "",
	}
}

func (m *ApplicationMetricLevel) Tags() map[string]string {
	tags := make(map[string]string)
	tags[constant.IpKey] = m.Ip
	tags[constant.HostnameKey] = m.HostName
	tags[constant.ApplicationKey] = m.ApplicationName
	tags[constant.ApplicationVersionKey] = m.Version
	tags[constant.GitCommitIdKey] = m.GitCommitId
	return tags
}

type ServiceMetricLevel struct {
	*ApplicationMetricLevel
	Interface string
}

func NewServiceMetric(interfaceName string) *ServiceMetricLevel {
	return &ServiceMetricLevel{ApplicationMetricLevel: GetApplicationLevel(), Interface: interfaceName}
}

func (m ServiceMetricLevel) Tags() map[string]string {
	tags := m.ApplicationMetricLevel.Tags()
	tags[constant.InterfaceKey] = m.Interface
	return tags
}

type MethodMetricLevel struct {
	*ServiceMetricLevel
	Method  string
	Group   string
	Version string
}

func (m MethodMetricLevel) Tags() map[string]string {
	tags := m.ServiceMetricLevel.Tags()
	tags[constant.MethodKey] = m.Method
	tags[constant.GroupKey] = m.Group
	tags[constant.VersionKey] = m.Version
	return tags
}

type ConfigCenterLevel struct {
	ApplicationName string
	Ip              string
	HostName        string
	Key             string
	Group           string
	ConfigCenter    string
	ChangeType      string
}

func NewConfigCenterLevel(key string, group string, configCenter string, changeType string) *ConfigCenterLevel {
	return &ConfigCenterLevel{
		ApplicationName: applicationName,
		Ip:              common.GetLocalIp(),
		HostName:        common.GetLocalHostName(),
		Key:             key,
		Group:           group,
		ConfigCenter:    configCenter,
		ChangeType:      changeType,
	}
}

func (l ConfigCenterLevel) Tags() map[string]string {
	tags := make(map[string]string)
	tags[constant.ApplicationKey] = l.ApplicationName
	tags[constant.IpKey] = l.Ip
	tags[constant.HostnameKey] = l.HostName
	tags[constant.KeyKey] = l.Key
	tags[constant.GroupKey] = l.Group
	tags[constant.ConfigCenterKey] = l.ConfigCenter
	tags[constant.ChangeTypeKey] = l.ChangeType
	return tags
}
