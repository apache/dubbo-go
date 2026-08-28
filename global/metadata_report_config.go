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

package global

import (
	"maps"
)

// MetadataReportConfig is app level configuration
type MetadataReportConfig struct {
	Protocol  string            `required:"true"  yaml:"protocol"  json:"protocol,omitempty"`
	Address   string            `required:"true" yaml:"address" json:"address"`
	Username  string            `yaml:"username" json:"username,omitempty"`
	Password  string            `yaml:"password" json:"password,omitempty"`
	Timeout   string            `yaml:"timeout" json:"timeout,omitempty"`
	Group     string            `yaml:"group" json:"group,omitempty"`
	Namespace string            `yaml:"namespace" json:"namespace,omitempty"`
	Params    map[string]string `yaml:"params"  json:"parameters,omitempty"`

	// ReportDefinition toggles publishing interface-level service definitions to
	// this report, which is what makes services visible to Dubbo Admin's
	// generic-invocation console.
	//
	// nil means enabled. This mirrors Java's MetadataReportConfig.reportDefinition,
	// which is the same switch with the same default. It is a pointer so an
	// explicit `false` is distinguishable from an unset field; with a plain bool
	// the zero value would silently disable the feature for everyone who does
	// not mention it.
	//
	// Turning this off is worth considering for a provider whose application,
	// version or group carries a per-deploy value, since each distinct
	// combination becomes a separate key that nothing ever deletes.
	ReportDefinition *bool `yaml:"report-definition" json:"report-definition,omitempty"`
}

func DefaultMetadataReportConfig() *MetadataReportConfig {
	// return a new config without setting any field means there is not any default value for initialization
	return &MetadataReportConfig{Params: map[string]string{}}
}

// Clone a new MetadataReportConfig
func (c *MetadataReportConfig) Clone() *MetadataReportConfig {
	if c == nil {
		return nil
	}

	newParams := make(map[string]string, len(c.Params))
	maps.Copy(newParams, c.Params)

	var reportDefinition *bool
	if c.ReportDefinition != nil {
		value := *c.ReportDefinition
		reportDefinition = &value
	}

	return &MetadataReportConfig{
		Protocol:         c.Protocol,
		Address:          c.Address,
		Username:         c.Username,
		Password:         c.Password,
		Timeout:          c.Timeout,
		Group:            c.Group,
		Namespace:        c.Namespace,
		Params:           newParams,
		ReportDefinition: reportDefinition,
	}
}
