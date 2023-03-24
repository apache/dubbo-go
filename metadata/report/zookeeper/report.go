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
	"encoding/json"
	"strings"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"

	gxset "github.com/dubbogo/gost/container/set"
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/metadata/report/factory"
)

var emptyStrSlice = make([]string, 0)

func init() {
	mf := &zookeeperMetadataReportFactory{}
	extension.SetMetadataReportFactory("zookeeper", func() factory.MetadataReportFactory {
		return mf
	})
}

// zookeeperMetadataReport is the implementation of
// MetadataReport based on zookeeper.
type zookeeperMetadataReport struct {
	client  *gxzookeeper.ZookeeperClient
	rootDir string
}

// GetAppMetadata get metadata info from zookeeper
func (m *zookeeperMetadataReport) GetAppMetadata(metadataIdentifier *identifier.SubscriberMetadataIdentifier) (*common.MetadataInfo, error) {
	k := m.rootDir + metadataIdentifier.GetFilePathKey()
	data, _, err := m.client.GetContent(k)
	if err != nil {
		return nil, err
	}
	var metadataInfo common.MetadataInfo
	err = json.Unmarshal(data, &metadataInfo)
	if err != nil {
		return nil, err
	}
	return &metadataInfo, nil
}

// PublishAppMetadata publish metadata info to zookeeper
func (m *zookeeperMetadataReport) PublishAppMetadata(metadataIdentifier *identifier.SubscriberMetadataIdentifier, info *common.MetadataInfo) error {
	k := m.rootDir + metadataIdentifier.GetFilePathKey()
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	err = m.client.CreateWithValue(k, data)
	if err == zk.ErrNodeExists {
		logger.Debugf("Try to create the node data failed. In most cases, it's not a problem. ")
		return nil
	}
	return err
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

// RegisterServiceAppMapping map the specified Dubbo service interface to current Dubbo app name
func (m *zookeeperMetadataReport) RegisterServiceAppMapping(key string, group string, value string) error {
	path := m.rootDir + group + constant.PathSeparator + key
	v, state, err := m.client.GetContent(path)
	if err == zk.ErrNoNode {
		return m.client.CreateWithValue(path, []byte(value))
	} else if err != nil {
		return err
	}
	oldValue := string(v)
	if strings.Contains(oldValue, value) {
		return nil
	}
	value = oldValue + constant.CommaSeparator + value
	_, err = m.client.SetContent(path, []byte(value), state.Version)
	return err
}

// GetServiceAppMapping get the app names from the specified Dubbo service interface
func (m *zookeeperMetadataReport) GetServiceAppMapping(key string, group string) (*gxset.HashSet, error) {
	path := m.rootDir + group + constant.PathSeparator + key
	v, _, err := m.client.GetContent(path)
	if err != nil {
		return nil, err
	}
	appNames := strings.Split(string(v), constant.CommaSeparator)
	set := gxset.NewSet()
	for _, e := range appNames {
		set.Add(e)
	}
	return set, nil
}

type zookeeperMetadataReportFactory struct{}

// nolint
func (mf *zookeeperMetadataReportFactory) CreateMetadataReport(url *common.URL) report.MetadataReport {
	client, err := gxzookeeper.NewZookeeperClient(
		"zookeeperMetadataReport",
		strings.Split(url.Location, ","),
		false,
		gxzookeeper.WithZkTimeOut(url.GetParamDuration(constant.TimeoutKey, "15s")),
	)
	if err != nil {
		panic(err)
	}

	rootDir := url.GetParam(constant.MetadataReportGroupKey, "dubbo")
	if !strings.HasPrefix(rootDir, constant.PathSeparator) {
		rootDir = constant.PathSeparator + rootDir
	}
	if rootDir != constant.PathSeparator {
		rootDir = rootDir + constant.PathSeparator
	}

	return &zookeeperMetadataReport{client: client, rootDir: rootDir}
}
