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
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/report/instance"
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
	// metadataType of this application is defined by application config, local or remote
	metadataType string
}

// Prefix dubbo.consumer
func (MetadataReportConfig) Prefix() string {
	return constant.MetadataReportPrefix
}

func (mc *MetadataReportConfig) Init(rc *RootConfig) error {
	if mc == nil {
		return nil
	}
	mc.metadataType = rc.Application.MetadataType
	if mc.Address != "" {
		tmpUrl, err := mc.ToUrl()
		if err != nil {
			logger.Errorf("MetadataReport init failed, err: %v", err)
			return err
		}
		// if metadata report config is avaible, then init metadata report instance
		instance.Init([]*common.URL{tmpUrl}, mc.metadataType)
		return nil
	}
	if rc.Registries != nil && len(rc.Registries) > 0 {
		// if metadata report config is not avaible, then init metadata report instance with registries
		urls := make([]*common.URL, 0)
		for id, reg := range rc.Registries {
			if reg.UseAsMetaReport && isValid(reg.Address) {
				if tmpUrl, err := reg.toMetadataReportUrl(); err == nil {
					tmpUrl.AddParam(constant.RegistryKey, id)
					urls = append(urls, tmpUrl)
				} else {
					logger.Warnf("use registry as metadata report failed, err: %v", err)
				}
			}
		}
		if len(urls) > 0 {
			instance.Init(urls, mc.metadataType)
		} else {
			logger.Warnf("No metadata report config found, skip init metadata report instance.")
		}
	}
	return nil
}

func (mc *MetadataReportConfig) ToUrl() (*common.URL, error) {
	res, err := common.NewURL(mc.Address,
		common.WithUsername(mc.Username),
		common.WithPassword(mc.Password),
		common.WithLocation(mc.Address),
		common.WithProtocol(mc.Protocol),
		common.WithParamsValue(constant.TimeoutKey, mc.Timeout),
		common.WithParamsValue(constant.MetadataReportGroupKey, mc.Group),
		common.WithParamsValue(constant.MetadataReportNamespaceKey, mc.Namespace),
		common.WithParamsValue(constant.MetadataTypeKey, mc.metadataType),
		common.WithParamsValue(constant.ClientNameKey, clientNameID(mc, mc.Protocol, mc.Address)),
	)
	if err != nil || len(res.Protocol) == 0 {
		return nil, perrors.New("Invalid MetadataReport Config.")
	}
	res.SetParam("metadata", res.Protocol)
	for key, val := range mc.Params {
		res.SetParam(key, val)
	}
	return res, nil
}

type MetadataReportConfigBuilder struct {
	metadataReportConfig *MetadataReportConfig
}

func NewMetadataReportConfigBuilder() *MetadataReportConfigBuilder {
	return &MetadataReportConfigBuilder{metadataReportConfig: newEmptyMetadataReportConfig()}
}

func (mrcb *MetadataReportConfigBuilder) SetProtocol(protocol string) *MetadataReportConfigBuilder {
	mrcb.metadataReportConfig.Protocol = protocol
	return mrcb
}

func (mrcb *MetadataReportConfigBuilder) SetAddress(address string) *MetadataReportConfigBuilder {
	mrcb.metadataReportConfig.Address = address
	return mrcb
}

func (mrcb *MetadataReportConfigBuilder) SetUsername(username string) *MetadataReportConfigBuilder {
	mrcb.metadataReportConfig.Username = username
	return mrcb
}

func (mrcb *MetadataReportConfigBuilder) SetPassword(password string) *MetadataReportConfigBuilder {
	mrcb.metadataReportConfig.Password = password
	return mrcb
}

func (mrcb *MetadataReportConfigBuilder) SetTimeout(timeout string) *MetadataReportConfigBuilder {
	mrcb.metadataReportConfig.Timeout = timeout
	return mrcb
}

func (mrcb *MetadataReportConfigBuilder) SetGroup(group string) *MetadataReportConfigBuilder {
	mrcb.metadataReportConfig.Group = group
	return mrcb
}

func (mrcb *MetadataReportConfigBuilder) Build() *MetadataReportConfig {
	return mrcb.metadataReportConfig
}

func newEmptyMetadataReportConfig() *MetadataReportConfig {
	return &MetadataReportConfig{
		Params: make(map[string]string),
	}
}
