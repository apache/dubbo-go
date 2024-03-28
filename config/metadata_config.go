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
	"strconv"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata"
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

func initMetadata(rc *RootConfig) error {
	opts := metadata.NewOptions(
		metadata.WithAppName(rc.Application.Name),
		metadata.WithMetadataType(rc.Application.MetadataType),
		metadata.WithPort(getMetadataPort(rc)),
	)
	if err := opts.Init(); err != nil {
		return err
	}
	return nil
}

func getMetadataPort(rc *RootConfig) int {
	port := rc.Application.MetadataServicePort
	if port == "" {
		protocolConfig, ok := rootConfig.Protocols[constant.DefaultProtocol]
		if ok {
			port = protocolConfig.Port
		} else {
			logger.Warnf("[Metadata Service] Dubbo-go %s version's MetadataService only support dubbo protocol,"+
				"MetadataService will use random port",
				constant.Version)
		}
	}
	if port == "" {
		return 0
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		logger.Error("MetadataService port parse error %v, MetadataService will use random port", err)
		return 0
	}
	return p
}

// Prefix dubbo.consumer
func (*MetadataReportConfig) Prefix() string {
	return constant.MetadataReportPrefix
}

func (mc *MetadataReportConfig) toReportOptions() (*metadata.ReportOptions, error) {
	opts := metadata.NewReportOptions(
		metadata.WithRegistryId(constant.DefaultKey),
		metadata.WithProtocol(mc.Protocol),
		metadata.WithAddress(mc.Address),
		metadata.WithUsername(mc.Username),
		metadata.WithPassword(mc.Password),
		metadata.WithGroup(mc.Group),
		metadata.WithNamespace(mc.Namespace),
		metadata.WithParams(mc.Params),
	)
	if mc.Timeout != "" {
		timeout, err := time.ParseDuration(mc.Timeout)
		if err != nil {
			return nil, err
		}
		metadata.WithTimeout(timeout)(opts)
	}
	return opts, nil
}

func registryToReportOptions(id string, rc *RegistryConfig) (*metadata.ReportOptions, error) {
	opts := metadata.NewReportOptions(
		metadata.WithRegistryId(id),
		metadata.WithProtocol(rc.Protocol),
		metadata.WithAddress(rc.Address),
		metadata.WithUsername(rc.Username),
		metadata.WithPassword(rc.Password),
		metadata.WithGroup(rc.Group),
		metadata.WithNamespace(rc.Namespace),
		metadata.WithParams(rc.Params),
	)
	if rc.Timeout != "" {
		timeout, err := time.ParseDuration(rc.Timeout)
		if err != nil {
			return nil, err
		}
		metadata.WithTimeout(timeout)(opts)
	}
	return opts, nil
}

func (mc *MetadataReportConfig) Init(rc *RootConfig) error {
	if mc == nil {
		return nil
	}
	mc.metadataType = rc.Application.MetadataType
	if mc.Address != "" {
		// if metadata report config is avaible, then init metadata report instance
		opts, err := mc.toReportOptions()
		if err != nil {
			return err
		}
		return opts.Init()
	}
	if rc.Registries != nil && len(rc.Registries) > 0 {
		// if metadata report config is not avaible, then init metadata report instance with registries
		for id, reg := range rc.Registries {
			if reg.UseAsMetaReport && isValid(reg.Address) {
				opts, err := registryToReportOptions(id, reg)
				if err != nil {
					return err
				}
				if err = opts.Init(); err != nil {
					return err
				}
			}
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
