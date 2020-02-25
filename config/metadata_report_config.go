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
)

// MethodConfig ...
type MetadataReportConfig struct {
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
		return err
	}
	type plain MetadataReportConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// ToUrl ...
func (c *MetadataReportConfig) ToUrl() (*common.URL, error) {
	urlMap := url.Values{}

	for k, v := range c.Params {
		urlMap.Set(k, v)
	}

	url, err := common.NewURL(c.Address,
		common.WithParams(urlMap),
		common.WithUsername(c.Username),
		common.WithPassword(c.Password),
		common.WithLocation(c.Address),
	)
	if err != nil {
		return nil, perrors.New("Invalid MetadataReportConfig.")
	}
	url.SetParam("metadata", url.Protocol)
	return &url, nil
}
