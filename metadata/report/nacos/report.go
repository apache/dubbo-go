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

	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/vo"
	perrors "github.com/pkg/errors"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/metadata/identifier"
)

// nacosMetadataReport is the implementation of MetadataReport based Nacos
type nacosMetadataReport struct {
	client config_client.IConfigClient
}

// StoreProviderMetadata will store the metadata
func (n *nacosMetadataReport) StoreProviderMetadata(providerIdentifier *identifier.MetadataIdentifier, serviceDefinitions string) error {
	return n.storeMetadata(vo.ConfigParam{
		DataId:  providerIdentifier.GetIdentifierKey(),
		Group:   providerIdentifier.Group,
		Content: serviceDefinitions,
	})
}

// StoreConsumerMetadata will store the metadata
func (n *nacosMetadataReport) StoreConsumerMetadata(consumerMetadataIdentifier *identifier.MetadataIdentifier, serviceParameterString string) error {
	return n.storeMetadata(vo.ConfigParam{
		DataId:  consumerMetadataIdentifier.GetIdentifierKey(),
		Group:   consumerMetadataIdentifier.Group,
		Content: serviceParameterString,
	})
}

// SaveServiceMetadata will store the metadata
func (n *nacosMetadataReport) SaveServiceMetadata(metadataIdentifier *identifier.ServiceMetadataIdentifier, url common.URL) error {
	return n.storeMetadata(vo.ConfigParam{
		DataId:  metadataIdentifier.GetIdentifierKey(),
		Group:   metadataIdentifier.Group,
		Content: url.String(),
	})
}

// RemoveServiceMetadata will remove the service metadata
func (n *nacosMetadataReport) RemoveServiceMetadata(metadataIdentifier *identifier.ServiceMetadataIdentifier) error {
	return n.deleteMetadata(vo.ConfigParam{
		DataId: metadataIdentifier.GetIdentifierKey(),
		Group:  metadataIdentifier.Group,
	})
}

// GetExportedURLs will look up the exported urls.
// if not found, an empty list will be returned.
func (n *nacosMetadataReport) GetExportedURLs(metadataIdentifier *identifier.ServiceMetadataIdentifier) []string {
	return n.getConfigAsArray(vo.ConfigParam{
		DataId: metadataIdentifier.GetIdentifierKey(),
		Group:  metadataIdentifier.Group,
	})
}

// SaveSubscribedData will convert the urlList to json array and then store it
func (n *nacosMetadataReport) SaveSubscribedData(subscriberMetadataIdentifier *identifier.SubscriberMetadataIdentifier, urlList []common.URL) error {
	if len(urlList) == 0 {
		logger.Warnf("The url list is empty")
		return nil
	}
	urlStrList := make([]string, 0, len(urlList))

	for _, e := range urlList {
		urlStrList = append(urlStrList, e.String())
	}

	bytes, err := json.Marshal(urlStrList)

	if err != nil {
		return perrors.WithMessage(err, "Could not convert the array to json")
	}
	return n.storeMetadata(vo.ConfigParam{
		DataId:  subscriberMetadataIdentifier.GetIdentifierKey(),
		Group:   subscriberMetadataIdentifier.Group,
		Content: string(bytes),
	})
}

// GetSubscribedURLs will lookup the url
// if not found, an empty list will be returned
func (n *nacosMetadataReport) GetSubscribedURLs(subscriberMetadataIdentifier *identifier.SubscriberMetadataIdentifier) []string {
	return n.getConfigAsArray(vo.ConfigParam{
		DataId: subscriberMetadataIdentifier.GetIdentifierKey(),
		Group:  subscriberMetadataIdentifier.Group,
	})
}

// GetServiceDefinition will lookup the service definition
func (n *nacosMetadataReport) GetServiceDefinition(metadataIdentifier *identifier.MetadataIdentifier) string {
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
func (n *nacosMetadataReport) getConfigAsArray(param vo.ConfigParam) []string {
	cfg := n.getConfig(param)
	res := make([]string, 0, 1)
	if len(cfg) == 0 {
		return res
	}
	decodeCfg, err := url.QueryUnescape(cfg)
	if err != nil {
		logger.Errorf("The config is invalid: %s", cfg)
	}
	res = append(res, decodeCfg)
	return res
}

// getConfig will read the config
func (n *nacosMetadataReport) getConfig(param vo.ConfigParam) string {
	cfg, err := n.client.GetConfig(param)
	if err != nil {
		logger.Errorf("Finding the configuration failed: %v", param)
	}
	return cfg
}
