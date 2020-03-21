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
	"strings"
	"sync"
)

import (
	"github.com/nacos-group/nacos-sdk-go/vo"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/config_center/parser"
)

const nacosClientName = "nacos config_center"

type nacosDynamicConfiguration struct {
	url          *common.URL
	rootPath     string
	wg           sync.WaitGroup
	cltLock      sync.Mutex
	done         chan struct{}
	client       *NacosClient
	keyListeners sync.Map
	parser       parser.ConfigurationParser
}

func newNacosDynamicConfiguration(url *common.URL) (*nacosDynamicConfiguration, error) {
	c := &nacosDynamicConfiguration{
		rootPath: "/" + url.GetParam(constant.CONFIG_NAMESPACE_KEY, config_center.DEFAULT_GROUP) + "/config",
		url:      url,
		done:     make(chan struct{}),
	}
	err := ValidateNacosClient(c, WithNacosName(nacosClientName))
	if err != nil {
		logger.Errorf("nacos client start error ,error message is %v", err)
		return nil, err
	}
	c.wg.Add(1)
	go HandleClientRestart(c)
	return c, err

}

// AddListener Add listener
func (nacos *nacosDynamicConfiguration) AddListener(key string, listener config_center.ConfigurationListener, opions ...config_center.Option) {
	nacos.addListener(key, listener)
}

// RemoveListener Remove listener
func (nacos *nacosDynamicConfiguration) RemoveListener(key string, listener config_center.ConfigurationListener, opions ...config_center.Option) {
	nacos.removeListener(key, listener)
}

// GetProperties nacos distinguishes configuration files based on group and dataId. defalut group = "dubbo" and dataId = key
func (nacos *nacosDynamicConfiguration) GetProperties(key string, opts ...config_center.Option) (string, error) {
	return nacos.GetRule(key, opts...)
}

// GetInternalProperty Get properties value by key
func (nacos *nacosDynamicConfiguration) GetInternalProperty(key string, opts ...config_center.Option) (string, error) {
	return nacos.GetProperties(key, opts...)
}

// PublishConfig will publish the config with the (key, group, value) pair
func (nacos *nacosDynamicConfiguration) PublishConfig(key string, group string, value string) error {

	group = nacos.resolvedGroup(group)

	ok, err := (*nacos.client.Client()).PublishConfig(vo.ConfigParam{
		DataId:  key,
		Group:   group,
		Content: value,
	})

	if err != nil {
		return err
	}
	if !ok {
		return perrors.New("publish config to Nocos failed")
	}
	return nil
}

// GetRule Get router rule
func (nacos *nacosDynamicConfiguration) GetRule(key string, opts ...config_center.Option) (string, error) {
	tmpOpts := &config_center.Options{}
	for _, opt := range opts {
		opt(tmpOpts)
	}
	content, err := (*nacos.client.Client()).GetConfig(vo.ConfigParam{
		DataId: key,
		Group:  nacos.resolvedGroup(tmpOpts.Group),
	})
	if err != nil {
		return "", perrors.WithStack(err)
	} else {
		return content, nil
	}
}

// Parser Get Parser
func (nacos *nacosDynamicConfiguration) Parser() parser.ConfigurationParser {
	return nacos.parser
}

// SetParser Set Parser
func (nacos *nacosDynamicConfiguration) SetParser(p parser.ConfigurationParser) {
	nacos.parser = p
}

// NacosClient Get Nacos Client
func (nacos *nacosDynamicConfiguration) NacosClient() *NacosClient {
	return nacos.client
}

// SetNacosClient Set Nacos Client
func (nacos *nacosDynamicConfiguration) SetNacosClient(client *NacosClient) {
	nacos.cltLock.Lock()
	nacos.client = client
	nacos.cltLock.Unlock()
}

// WaitGroup for wait group control, zk client listener & zk client container
func (nacos *nacosDynamicConfiguration) WaitGroup() *sync.WaitGroup {
	return &nacos.wg
}

// GetDone For nacos client control	RestartCallBack() bool
func (nacos *nacosDynamicConfiguration) GetDone() chan struct{} {
	return nacos.done
}

// GetUrl Get Url
func (nacos *nacosDynamicConfiguration) GetUrl() common.URL {
	return *nacos.url
}

// Destroy Destroy configuration instance
func (nacos *nacosDynamicConfiguration) Destroy() {
	close(nacos.done)
	nacos.wg.Wait()
	nacos.closeConfigs()
}

// resolvedGroup will regular the group. Now, it will replace the '/' with '-'.
// '/' is a special character for nacos
func (nacos *nacosDynamicConfiguration) resolvedGroup(group string) string {
	if len(group) <= 0 {
		return group
	}
	return strings.ReplaceAll(group, "/", "-")
}

// IsAvailable Get available status
func (nacos *nacosDynamicConfiguration) IsAvailable() bool {
	select {
	case <-nacos.done:
		return false
	default:
		return true
	}
}

func (nacos *nacosDynamicConfiguration) closeConfigs() {
	nacos.cltLock.Lock()
	client := nacos.client
	nacos.client = nil
	nacos.cltLock.Unlock()
	// Close the old client first to close the tmp node
	client.Close()
	logger.Infof("begin to close provider nacos client")
}
