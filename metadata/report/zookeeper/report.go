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

package zookeeper

import (
	"strings"
	"time"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/metadata/identifier"
	"github.com/apache/dubbo-go/metadata/report"
	"github.com/apache/dubbo-go/metadata/report/factory"
	"github.com/apache/dubbo-go/remoting/zookeeper"
)

var (
	emptyStrSlice = make([]string, 0)
)

func init() {
	mf := &zookeeperMetadataReportFactory{}
	extension.SetMetadataReportFactory("zookeeper", func() factory.MetadataReportFactory {
		return mf
	})
}

// zookeeperMetadataReport is the implementation of
// MetadataReport based on zookeeper.
type zookeeperMetadataReport struct {
	client  *zookeeper.ZookeeperClient
	rootDir string
}

// StoreProviderMetadata stores the metadata.
func (m *zookeeperMetadataReport) StoreProviderMetadata(providerIdentifier *identifier.MetadataIdentifier, serviceDefinitions string) error {
	k := m.rootDir + providerIdentifier.GetFilePathKey()
	return m.client.CreateWithValue(k, []byte(serviceDefinitions))
}

// StoreConsumerMetadata stores the metadata.
func (m *zookeeperMetadataReport) StoreConsumerMetadata(consumerMetadataIdentifier *identifier.MetadataIdentifier, serviceParameterString string) error {
	k := m.rootDir + consumerMetadataIdentifier.GetFilePathKey()
	return m.client.CreateWithValue(k, []byte(serviceParameterString))
}

// SaveServiceMetadata saves the metadata.
func (m *zookeeperMetadataReport) SaveServiceMetadata(metadataIdentifier *identifier.ServiceMetadataIdentifier, url *common.URL) error {
	k := m.rootDir + metadataIdentifier.GetFilePathKey()
	return m.client.CreateWithValue(k, []byte(url.String()))
}

// RemoveServiceMetadata removes the metadata.
func (m *zookeeperMetadataReport) RemoveServiceMetadata(metadataIdentifier *identifier.ServiceMetadataIdentifier) error {
	k := m.rootDir + metadataIdentifier.GetFilePathKey()
	return m.client.Delete(k)
}

// GetExportedURLs gets the urls.
func (m *zookeeperMetadataReport) GetExportedURLs(metadataIdentifier *identifier.ServiceMetadataIdentifier) ([]string, error) {
	k := m.rootDir + metadataIdentifier.GetFilePathKey()
	v, _, err := m.client.GetContent(k)
	if err != nil || len(v) == 0 {
		return emptyStrSlice, err
	}
	return []string{string(v)}, nil
}

// SaveSubscribedData saves the urls.
func (m *zookeeperMetadataReport) SaveSubscribedData(subscriberMetadataIdentifier *identifier.SubscriberMetadataIdentifier, urls string) error {
	k := m.rootDir + subscriberMetadataIdentifier.GetFilePathKey()
	return m.client.CreateWithValue(k, []byte(urls))
}

// GetSubscribedURLs gets the urls.
func (m *zookeeperMetadataReport) GetSubscribedURLs(subscriberMetadataIdentifier *identifier.SubscriberMetadataIdentifier) ([]string, error) {
	k := m.rootDir + subscriberMetadataIdentifier.GetFilePathKey()
	v, _, err := m.client.GetContent(k)
	if err != nil || len(v) == 0 {
		return emptyStrSlice, err
	}
	return []string{string(v)}, nil
}

// GetServiceDefinition gets the service definition.
func (m *zookeeperMetadataReport) GetServiceDefinition(metadataIdentifier *identifier.MetadataIdentifier) (string, error) {
	k := m.rootDir + metadataIdentifier.GetFilePathKey()
	v, _, err := m.client.GetContent(k)
	return string(v), err
}

type zookeeperMetadataReportFactory struct {
}

// nolint
func (mf *zookeeperMetadataReportFactory) CreateMetadataReport(url *common.URL) report.MetadataReport {
	client, err := zookeeper.NewZookeeperClient(
		"zookeeperMetadataReport",
		strings.Split(url.Location, ","),
		15*time.Second,
	)
	if err != nil {
		panic(err)
	}

	rootDir := url.GetParam(constant.GROUP_KEY, "dubbo")
	if !strings.HasPrefix(rootDir, constant.PATH_SEPARATOR) {
		rootDir = constant.PATH_SEPARATOR + rootDir
	}
	if rootDir != constant.PATH_SEPARATOR {
		rootDir = rootDir + constant.PATH_SEPARATOR
	}

	return &zookeeperMetadataReport{client: client, rootDir: rootDir}
}
