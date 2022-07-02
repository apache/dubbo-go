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
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config/instance"
)

// MetadataReportConfig is app level configuration
type MetadataReportConfig struct {
	Protocol  string `required:"true"  yaml:"protocol"  json:"protocol,omitempty"`
	Address   string `required:"true" yaml:"address" json:"address"`
	Username  string `yaml:"username" json:"username,omitempty"`
	Password  string `yaml:"password" json:"password,omitempty"`
	Timeout   string `yaml:"timeout" json:"timeout,omitempty"`
	Group     string `yaml:"group" json:"group,omitempty"`
	Namespace string `yaml:"namespace" json:"namespace,omitempty"`
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
	return mc.StartMetadataReport()
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
	return res, nil
}

func (mc *MetadataReportConfig) IsValid() bool {
	return len(mc.Protocol) != 0
}

// StartMetadataReport  The entry of metadata report start
func (mc *MetadataReportConfig) StartMetadataReport() error {
	if mc == nil || !mc.IsValid() {
		return nil
	}
	if tmpUrl, err := mc.ToUrl(); err == nil {
		instance.GetMetadataReportInstance(tmpUrl)
		return nil
	} else {
		return perrors.Wrap(err, "Start MetadataReport failed.")
	}
}

func publishServiceDefinition(url *common.URL) {
	localService, err := extension.GetLocalMetadataService(constant.DefaultKey)
	if err != nil {
		logger.Warnf("get local metadata service failed, please check if you have imported _ \"dubbo.apache.org/dubbo-go/v3/metadata/service/local\"")
		return
	}
	localService.PublishServiceDefinition(url)
	if url.GetParam(constant.MetadataTypeKey, "") != constant.RemoteMetadataStorageType {
		return
	}
	if remoteMetadataService, err := extension.GetRemoteMetadataService(); err == nil && remoteMetadataService != nil {
		remoteMetadataService.PublishServiceDefinition(url)
	}
}

//
// selectMetadataServiceExportedURL get already be exported url
func selectMetadataServiceExportedURL() *common.URL {
	var selectedUrl *common.URL
	metaDataService, err := extension.GetLocalMetadataService(constant.DefaultKey)
	if err != nil {
		logger.Warnf("get metadata service exporter failed, pls check if you import _ \"dubbo.apache.org/dubbo-go/v3/metadata/service/local\"")
		return nil
	}
	urlList, err := metaDataService.GetExportedURLs(constant.AnyValue, constant.AnyValue, constant.AnyValue, constant.AnyValue)
	if err != nil {
		panic(err)
	}
	if len(urlList) == 0 {
		return nil
	}
	for _, url := range urlList {
		selectedUrl = url
		// rest first
		if url.Protocol == "rest" {
			break
		}
	}
	return selectedUrl
}

type MetadataReportConfigBuilder struct {
	metadataReportConfig *MetadataReportConfig
}

func NewMetadataReportConfigBuilder() *MetadataReportConfigBuilder {
	return &MetadataReportConfigBuilder{metadataReportConfig: &MetadataReportConfig{}}
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
