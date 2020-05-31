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

package config

import (
	"net/url"
)

import (
	"github.com/creasty/defaults"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/config/instance"
)

// MethodConfig ...
type MetadataReportConfig struct {
	Protocol   string            `required:"true"  yaml:"protocol"  json:"protocol,omitempty"`
	Address    string            `yaml:"address" json:"address,omitempty" property:"address"`
	Username   string            `yaml:"username" json:"username,omitempty" property:"username"`
	Password   string            `yaml:"password" json:"password,omitempty"  property:"password"`
	Params     map[string]string `yaml:"params" json:"params,omitempty" property:"params"`
	TimeoutStr string            `yaml:"timeout" default:"5s" json:"timeout,omitempty" property:"timeout"` // unit: second
	Group      string            `yaml:"group" json:"group,omitempty" property:"group"`
}

// Prefix ...
func (c *MetadataReportConfig) Prefix() string {
	return constant.MetadataReportPrefix
}

// UnmarshalYAML ...
func (c *MetadataReportConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return perrors.WithStack(err)
	}
	type plain MetadataReportConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return perrors.WithStack(err)
	}
	return nil
}

// ToUrl ...
func (c *MetadataReportConfig) ToUrl() (*common.URL, error) {
	urlMap := make(url.Values)

	if c.Params != nil {
		for k, v := range c.Params {
			urlMap.Set(k, v)
		}
	}

	url, err := common.NewURL(c.Address,
		common.WithParams(urlMap),
		common.WithUsername(c.Username),
		common.WithPassword(c.Password),
		common.WithLocation(c.Address),
		common.WithProtocol(c.Protocol),
	)
	if err != nil || len(url.Protocol) == 0 {
		return nil, perrors.New("Invalid MetadataReportConfig.")
	}
	url.SetParam("metadata", url.Protocol)
	return &url, nil
}

func (c *MetadataReportConfig) IsValid() bool {
	return len(c.Protocol) != 0
}

// StartMetadataReport: The entry of metadata report start
func startMetadataReport(metadataType string, metadataReportConfig *MetadataReportConfig) error {
	if metadataReportConfig == nil || metadataReportConfig.IsValid() {
		return nil
	}

	if metadataType == constant.METACONFIG_REMOTE {
		return perrors.New("No MetadataConfig found, you must specify the remote Metadata Center address when 'metadata=remote' is enabled.")
	} else if metadataType == constant.METACONFIG_REMOTE && len(metadataReportConfig.Address) == 0 {
		return perrors.New("MetadataConfig address can not be empty.")
	}

	if url, err := metadataReportConfig.ToUrl(); err == nil {
		instance.GetMetadataReportInstance(url)
	} else {
		return perrors.Wrap(err, "Start MetadataReport failed.")
	}

	return nil
}
