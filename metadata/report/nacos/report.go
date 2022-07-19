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

package nacos

import (
	"encoding/json"
	"net/url"
	"strings"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
	"github.com/dubbogo/gost/log/logger"

	"github.com/nacos-group/nacos-sdk-go/vo"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metadata/report/factory"
	"dubbo.apache.org/dubbo-go/v3/remoting/nacos"
)

func init() {
	mf := &nacosMetadataReportFactory{}
	extension.SetMetadataReportFactory("nacos", func() factory.MetadataReportFactory {
		return mf
	})
}

// nacosMetadataReport is the implementation
// of MetadataReport based on nacos.
type nacosMetadataReport struct {
	client *nacosClient.NacosConfigClient
}

// GetAppMetadata get metadata info from nacos
func (n *nacosMetadataReport) GetAppMetadata(metadataIdentifier *identifier.SubscriberMetadataIdentifier) (*common.MetadataInfo, error) {
	data, err := n.getConfig(vo.ConfigParam{
		DataId: metadataIdentifier.GetIdentifierKey(),
		Group:  metadataIdentifier.Group,
	})
	if err != nil {
		return nil, err
	}

	var metadataInfo common.MetadataInfo
	err = json.Unmarshal([]byte(data), &metadataInfo)
	if err != nil {
		return nil, err
	}
	return &metadataInfo, nil
}

// PublishAppMetadata publish metadata info to nacos
func (n *nacosMetadataReport) PublishAppMetadata(metadataIdentifier *identifier.SubscriberMetadataIdentifier, info *common.MetadataInfo) error {
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	return n.storeMetadata(vo.ConfigParam{
		DataId:  metadataIdentifier.GetIdentifierKey(),
		Group:   metadataIdentifier.Group,
		Content: string(data),
	})
}

// StoreProviderMetadata stores the metadata.
func (n *nacosMetadataReport) StoreProviderMetadata(providerIdentifier *identifier.MetadataIdentifier, serviceDefinitions string) error {
	return n.storeMetadata(vo.ConfigParam{
		DataId:  providerIdentifier.GetIdentifierKey(),
		Group:   providerIdentifier.Group,
		Content: serviceDefinitions,
	})
}

// StoreConsumerMetadata stores the metadata.
func (n *nacosMetadataReport) StoreConsumerMetadata(consumerMetadataIdentifier *identifier.MetadataIdentifier, serviceParameterString string) error {
	return n.storeMetadata(vo.ConfigParam{
		DataId:  consumerMetadataIdentifier.GetIdentifierKey(),
		Group:   consumerMetadataIdentifier.Group,
		Content: serviceParameterString,
	})
}

// SaveServiceMetadata saves the metadata.
func (n *nacosMetadataReport) SaveServiceMetadata(metadataIdentifier *identifier.ServiceMetadataIdentifier, url *common.URL) error {
	return n.storeMetadata(vo.ConfigParam{
		DataId:  metadataIdentifier.GetIdentifierKey(),
		Group:   metadataIdentifier.Group,
		Content: url.String(),
	})
}

// RemoveServiceMetadata removes the metadata.
func (n *nacosMetadataReport) RemoveServiceMetadata(metadataIdentifier *identifier.ServiceMetadataIdentifier) error {
	return n.deleteMetadata(vo.ConfigParam{
		DataId: metadataIdentifier.GetIdentifierKey(),
		Group:  metadataIdentifier.Group,
	})
}

// GetExportedURLs gets the urls.
func (n *nacosMetadataReport) GetExportedURLs(metadataIdentifier *identifier.ServiceMetadataIdentifier) ([]string, error) {
	return n.getConfigAsArray(vo.ConfigParam{
		DataId: metadataIdentifier.GetIdentifierKey(),
		Group:  metadataIdentifier.Group,
	})
}

// SaveSubscribedData saves the urls.
func (n *nacosMetadataReport) SaveSubscribedData(subscriberMetadataIdentifier *identifier.SubscriberMetadataIdentifier, urls string) error {
	return n.storeMetadata(vo.ConfigParam{
		DataId:  subscriberMetadataIdentifier.GetIdentifierKey(),
		Content: urls,
	})
}

// GetSubscribedURLs gets the urls.
func (n *nacosMetadataReport) GetSubscribedURLs(subscriberMetadataIdentifier *identifier.SubscriberMetadataIdentifier) ([]string, error) {
	return n.getConfigAsArray(vo.ConfigParam{
		DataId: subscriberMetadataIdentifier.GetIdentifierKey(),
	})
}

// GetServiceDefinition gets the service definition.
func (n *nacosMetadataReport) GetServiceDefinition(metadataIdentifier *identifier.MetadataIdentifier) (string, error) {
	return n.getConfig(vo.ConfigParam{
		DataId: metadataIdentifier.GetIdentifierKey(),
		Group:  metadataIdentifier.Group,
	})
}

// storeMetadata will publish the metadata to Nacos
// if failed or error is not nil, error will be returned
func (n *nacosMetadataReport) storeMetadata(param vo.ConfigParam) error {
	res, err := n.client.Client().PublishConfig(param)
	if err != nil {
		return perrors.WithMessage(err, "Could not publish the metadata")
	}
	if !res {
		return perrors.New("Publish the metadata failed.")
	}
	return nil
}

// deleteMetadata will delete the metadata
func (n *nacosMetadataReport) deleteMetadata(param vo.ConfigParam) error {
	res, err := n.client.Client().DeleteConfig(param)
	if err != nil {
		return perrors.WithMessage(err, "Could not delete the metadata")
	}
	if !res {
		return perrors.New("Deleting the metadata failed.")
	}
	return nil
}

// getConfigAsArray will read the config and then convert it as an one-element array
// error or config not found, an empty list will be returned.
func (n *nacosMetadataReport) getConfigAsArray(param vo.ConfigParam) ([]string, error) {
	res := make([]string, 0, 1)

	cfg, err := n.getConfig(param)
	if err != nil || len(cfg) == 0 {
		return res, err
	}

	decodeCfg, err := url.QueryUnescape(cfg)
	if err != nil {
		logger.Errorf("The config is invalid: %s", cfg)
		return res, err
	}

	res = append(res, decodeCfg)
	return res, nil
}

// getConfig will read the config
func (n *nacosMetadataReport) getConfig(param vo.ConfigParam) (string, error) {
	cfg, err := n.client.Client().GetConfig(param)
	if err != nil {
		logger.Errorf("Finding the configuration failed: %v", param)
		return "", err
	}
	return cfg, nil
}

// RegisterServiceAppMapping map the specified Dubbo service interface to current Dubbo app name
func (n *nacosMetadataReport) RegisterServiceAppMapping(key string, group string, value string) error {
	oldVal, err := n.getConfig(vo.ConfigParam{
		DataId: key,
		Group:  group,
	})
	if err != nil {
		return err
	}
	if strings.Contains(oldVal, value) {
		return nil
	}
	if oldVal != "" {
		value = oldVal + constant.CommaSeparator + value
	}
	return n.storeMetadata(vo.ConfigParam{
		DataId:  key,
		Group:   group,
		Content: value,
	})
}

// GetServiceAppMapping get the app names from the specified Dubbo service interface
func (n *nacosMetadataReport) GetServiceAppMapping(key string, group string) (*gxset.HashSet, error) {
	v, err := n.getConfig(vo.ConfigParam{
		DataId: key,
		Group:  group,
	})
	if err != nil {
		return nil, err
	}
	if v == "" {
		return nil, perrors.New("There is no service app mapping data.")
	}
	appNames := strings.Split(v, constant.CommaSeparator)
	set := gxset.NewSet()
	for _, e := range appNames {
		set.Add(e)
	}
	return set, nil
}

type nacosMetadataReportFactory struct{}

// nolint
func (n *nacosMetadataReportFactory) CreateMetadataReport(url *common.URL) report.MetadataReport {
	url.SetParam(constant.NacosNamespaceID, url.GetParam(constant.MetadataReportNamespaceKey, ""))
	url.SetParam(constant.TimeoutKey, url.GetParam(constant.TimeoutKey, constant.DefaultRegTimeout))
	url.SetParam(constant.NacosGroupKey, url.GetParam(constant.MetadataReportGroupKey, constant.ServiceDiscoveryDefaultGroup))
	url.SetParam(constant.NacosUsername, url.Username)
	url.SetParam(constant.NacosPassword, url.Password)
	client, err := nacos.NewNacosConfigClientByUrl(url)
	if err != nil {
		logger.Errorf("Could not create nacos metadata report. URL: %s", url.String())
		return nil
	}
	return &nacosMetadataReport{client: client}
}
