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
	"fmt"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config/instance"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
)

// MetadataReportConfig is app level configuration
type MetadataReportConfig struct {
	Protocol string `required:"true"  yaml:"protocol"  json:"protocol,omitempty"`
	Address  string `required:"true" yaml:"address" json:"address"`
	Username string `yaml:"username" json:"username,omitempty"`
	Password string `yaml:"password" json:"password,omitempty"`
	Timeout  string `yaml:"timeout" json:"timeout,omitempty"`
	Group    string `yaml:"group" json:"group,omitempty"`

	MetadataReportType string
}

// Prefix dubbo.consumer
func (MetadataReportConfig) Prefix() string {
	return constant.MetadataReportPrefix
}

func (c *MetadataReportConfig) Init(rc *RootConfig) error {
	if c == nil {
		return nil
	}
	c.MetadataReportType = rc.Application.MetadataType
	return c.StartMetadataReport()
}

// nolint
func (c *MetadataReportConfig) ToUrl() (*common.URL, error) {
	res, err := common.NewURL(c.Address,
		//common.WithParams(urlMap),
		common.WithUsername(c.Username),
		common.WithPassword(c.Password),
		common.WithLocation(c.Address),
		common.WithProtocol(c.Protocol),
		common.WithParamsValue(constant.METADATATYPE_KEY, c.MetadataReportType),
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
func (c *MetadataReportConfig) StartMetadataReport() error {
	if c == nil || !c.IsValid() {
		return nil
	}
	if tmpUrl, err := c.ToUrl(); err == nil {
		instance.GetMetadataReportInstance(tmpUrl)
		return nil
	} else {
		return perrors.Wrap(err, "Start MetadataReport failed.")
	}
}

func publishServiceDefinition(url *common.URL) {
	if url.GetParam(constant.METADATATYPE_KEY, "") != constant.REMOTE_METADATA_STORAGE_TYPE {
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
	var mds service.MetadataService
	var err error
	if GetApplicationConfig().MetadataType == constant.DEFAULT_METADATA_STORAGE_TYPE {
		mds, err = extension.GetLocalMetadataService("")
	} else {
		mds, err = extension.GetRemoteMetadataService()
	}
	if err != nil {
		fmt.Println("selectMetadataServiceExportedURL err = ", err)
		return nil
	}

	urlList, err := mds.GetExportedURLs(constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE, constant.ANY_VALUE)
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
