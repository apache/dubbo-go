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

// MethodConfig is method level configuration
type MetadataReportConfig struct {
	Protocol  string            `required:"true"  yaml:"protocol"  json:"protocol,omitempty"`
	RemoteRef string            `required:"true"  yaml:"remote_ref"  json:"remote_ref,omitempty"`
	Params    map[string]string `yaml:"params" json:"params,omitempty" property:"params"`
	Group     string            `yaml:"group" json:"group,omitempty" property:"group"`
}

// nolint
func (c *MetadataReportConfig) Prefix() string {
	return constant.MetadataReportPrefix
}

// UnmarshalYAML unmarshal the MetadataReportConfig by @unmarshal function
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

// nolint
func (c *MetadataReportConfig) ToUrl() (*common.URL, error) {
	urlMap := make(url.Values)

	if c.Params != nil {
		for k, v := range c.Params {
			urlMap.Set(k, v)
		}
	}

	rc, ok := GetBaseConfig().GetRemoteConfig(c.RemoteRef)

	if !ok {
		return nil, perrors.New("Could not find out the remote ref config, name: " + c.RemoteRef)
	}

	res, err := common.NewURL(rc.Address,
		common.WithParams(urlMap),
		common.WithUsername(rc.Username),
		common.WithPassword(rc.Password),
		common.WithLocation(rc.Address),
		common.WithProtocol(c.Protocol),
	)
	if err != nil || len(res.Protocol) == 0 {
		return nil, perrors.New("Invalid MetadataReportConfig.")
	}
	res.SetParam("metadata", res.Protocol)
	return res, nil
}

func (c *MetadataReportConfig) IsValid() bool {
	return len(c.Protocol) != 0
}

// StartMetadataReport: The entry of metadata report start
func startMetadataReport(metadataType string, metadataReportConfig *MetadataReportConfig) error {
	if metadataReportConfig == nil || !metadataReportConfig.IsValid() {
		return nil
	}

	if metadataType == constant.METACONFIG_REMOTE && len(metadataReportConfig.RemoteRef) == 0 {
		return perrors.New("MetadataConfig remote ref can not be empty.")
	}

	if tmpUrl, err := metadataReportConfig.ToUrl(); err == nil {
		instance.GetMetadataReportInstance(tmpUrl)
	} else {
		return perrors.Wrap(err, "Start MetadataReport failed.")
	}

	return nil
}
