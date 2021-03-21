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

package consul

import (
	consul "github.com/hashicorp/consul/api"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/metadata/identifier"
	"github.com/apache/dubbo-go/metadata/report"
	"github.com/apache/dubbo-go/metadata/report/factory"
)

var (
	emptyStrSlice = make([]string, 0)
)

func init() {
	mf := &consulMetadataReportFactory{}
	extension.SetMetadataReportFactory("consul", func() factory.MetadataReportFactory {
		return mf
	})
}

// consulMetadataReport is the implementation of
// MetadataReport based on consul.
type consulMetadataReport struct {
	client *consul.Client
}

// StoreProviderMetadata stores the metadata.
func (m *consulMetadataReport) StoreProviderMetadata(providerIdentifier *identifier.MetadataIdentifier, serviceDefinitions string) error {
	kv := &consul.KVPair{Key: providerIdentifier.GetIdentifierKey(), Value: []byte(serviceDefinitions)}
	_, err := m.client.KV().Put(kv, nil)
	return err
}

// StoreConsumerMetadata stores the metadata.
func (m *consulMetadataReport) StoreConsumerMetadata(consumerMetadataIdentifier *identifier.MetadataIdentifier, serviceParameterString string) error {
	kv := &consul.KVPair{Key: consumerMetadataIdentifier.GetIdentifierKey(), Value: []byte(serviceParameterString)}
	_, err := m.client.KV().Put(kv, nil)
	return err
}

// SaveServiceMetadata saves the metadata.
func (m *consulMetadataReport) SaveServiceMetadata(metadataIdentifier *identifier.ServiceMetadataIdentifier, url *common.URL) error {
	kv := &consul.KVPair{Key: metadataIdentifier.GetIdentifierKey(), Value: []byte(url.String())}
	_, err := m.client.KV().Put(kv, nil)
	return err
}

// RemoveServiceMetadata removes the metadata.
func (m *consulMetadataReport) RemoveServiceMetadata(metadataIdentifier *identifier.ServiceMetadataIdentifier) error {
	k := metadataIdentifier.GetIdentifierKey()
	_, err := m.client.KV().Delete(k, nil)
	return err
}

// GetExportedURLs gets the urls.
func (m *consulMetadataReport) GetExportedURLs(metadataIdentifier *identifier.ServiceMetadataIdentifier) ([]string, error) {
	k := metadataIdentifier.GetIdentifierKey()
	kv, _, err := m.client.KV().Get(k, nil)
	if err != nil || kv == nil {
		return emptyStrSlice, err
	}
	return []string{string(kv.Value)}, nil
}

// SaveSubscribedData saves the urls.
func (m *consulMetadataReport) SaveSubscribedData(subscriberMetadataIdentifier *identifier.SubscriberMetadataIdentifier, urls string) error {
	kv := &consul.KVPair{Key: subscriberMetadataIdentifier.GetIdentifierKey(), Value: []byte(urls)}
	_, err := m.client.KV().Put(kv, nil)
	return err
}

// GetSubscribedURLs gets the urls.
func (m *consulMetadataReport) GetSubscribedURLs(subscriberMetadataIdentifier *identifier.SubscriberMetadataIdentifier) ([]string, error) {
	k := subscriberMetadataIdentifier.GetIdentifierKey()
	kv, _, err := m.client.KV().Get(k, nil)
	if err != nil || kv == nil {
		return emptyStrSlice, err
	}
	return []string{string(kv.Value)}, nil
}

// GetServiceDefinition gets the service definition.
func (m *consulMetadataReport) GetServiceDefinition(metadataIdentifier *identifier.MetadataIdentifier) (string, error) {
	k := metadataIdentifier.GetIdentifierKey()
	kv, _, err := m.client.KV().Get(k, nil)
	if err != nil || kv == nil {
		return "", err
	}
	return string(kv.Value), nil
}

type consulMetadataReportFactory struct {
}

// nolint
func (mf *consulMetadataReportFactory) CreateMetadataReport(url *common.URL) report.MetadataReport {
	config := &consul.Config{Address: url.Location}
	client, err := consul.NewClient(config)
	if err != nil {
		panic(err)
	}
	return &consulMetadataReport{client: client}
}
