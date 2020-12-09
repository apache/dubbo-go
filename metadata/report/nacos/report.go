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
	"net/url"
)

import (
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/metadata/identifier"
	"github.com/apache/dubbo-go/metadata/report"
	"github.com/apache/dubbo-go/metadata/report/factory"
	"github.com/apache/dubbo-go/remoting/nacos"
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
	client config_client.IConfigClient
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
		Group:   subscriberMetadataIdentifier.Group,
		Content: urls,
	})
}

// GetSubscribedURLs gets the urls.
func (n *nacosMetadataReport) GetSubscribedURLs(subscriberMetadataIdentifier *identifier.SubscriberMetadataIdentifier) ([]string, error) {
	return n.getConfigAsArray(vo.ConfigParam{
		DataId: subscriberMetadataIdentifier.GetIdentifierKey(),
		Group:  subscriberMetadataIdentifier.Group,
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
	res, err := n.client.PublishConfig(param)
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
	res, err := n.client.DeleteConfig(param)
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
	cfg, err := n.client.GetConfig(param)
	if err != nil {
		logger.Errorf("Finding the configuration failed: %v", param)
		return "", err
	}
	return cfg, nil
}

type nacosMetadataReportFactory struct {
}

// nolint
func (n *nacosMetadataReportFactory) CreateMetadataReport(url *common.URL) report.MetadataReport {
	client, err := nacos.NewNacosConfigClient(url)
	if err != nil {
		logger.Errorf("Could not create nacos metadata report. URL: %s", url.String())
		return nil
	}
	return &nacosMetadataReport{client: client}
}
