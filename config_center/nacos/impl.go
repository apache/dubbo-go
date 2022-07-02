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
	gxset "github.com/dubbogo/gost/container/set"
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
	"github.com/dubbogo/gost/log/logger"

	constant2 "github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
)

const (
	nacosClientName = "nacos config_center"
	// the number is a little big tricky
	// it will be used in query which looks up all keys with the target group
	// now, one key represents one application
	// so only a group has more than 9999 applications will failed
	maxKeysNum = 9999
)

// nacosDynamicConfiguration is the implementation of DynamicConfiguration based on nacos
type nacosDynamicConfiguration struct {
	config_center.BaseDynamicConfiguration
	url          *common.URL
	rootPath     string
	wg           sync.WaitGroup
	cltLock      sync.Mutex
	done         chan struct{}
	client       *nacosClient.NacosConfigClient
	keyListeners sync.Map
	parser       parser.ConfigurationParser
}

func newNacosDynamicConfiguration(url *common.URL) (*nacosDynamicConfiguration, error) {
	url.SetParam(constant.NacosNamespaceID, url.GetParam(constant.ConfigNamespaceKey, ""))
	url.SetParam(constant.NacosUsername, url.Username)
	url.SetParam(constant.NacosPassword, url.Password)
	url.SetParam(constant.NacosAccessKey, url.GetParam(constant.ConfigAccessKey, ""))
	url.SetParam(constant.NacosSecretKey, url.GetParam(constant.ConfigSecretKey, ""))
	url.SetParam(constant.NacosTimeout, url.GetParam(constant.ConfigTimeoutKey, ""))
	url.SetParam(constant.NacosGroupKey, url.GetParam(constant.ConfigGroupKey, constant2.DEFAULT_GROUP))
	c := &nacosDynamicConfiguration{
		url:  url,
		done: make(chan struct{}),
	}
	c.GetURL()
	logger.Infof("[Nacos ConfigCenter] New Nacos ConfigCenter with Configuration: %+v, url = %+v", c, c.GetURL())
	err := ValidateNacosClient(c)
	if err != nil {
		logger.Errorf("nacos configClient start error ,error message is %v", err)
		return nil, err
	}
	c.wg.Add(1)
	go HandleClientRestart(c)
	return c, err
}

// AddListener Add listener
func (n *nacosDynamicConfiguration) AddListener(key string, listener config_center.ConfigurationListener, opions ...config_center.Option) {
	n.addListener(key, listener)
}

// RemoveListener Remove listener
func (n *nacosDynamicConfiguration) RemoveListener(key string, listener config_center.ConfigurationListener, opions ...config_center.Option) {
	n.removeListener(key, listener)
}

// GetProperties nacos distinguishes configuration files based on group and dataId. defalut group = "dubbo" and dataId = key
func (n *nacosDynamicConfiguration) GetProperties(key string, opts ...config_center.Option) (string, error) {
	return n.GetRule(key, opts...)
}

// GetInternalProperty Get properties value by key
func (n *nacosDynamicConfiguration) GetInternalProperty(key string, opts ...config_center.Option) (string, error) {
	return n.GetProperties(key, opts...)
}

// PublishConfig will publish the config with the (key, group, value) pair
func (n *nacosDynamicConfiguration) PublishConfig(key string, group string, value string) error {
	group = n.resolvedGroup(group)
	ok, err := n.client.Client().PublishConfig(vo.ConfigParam{
		DataId:  key,
		Group:   group,
		Content: value,
	})
	if err != nil {
		return perrors.WithStack(err)
	}
	if !ok {
		return perrors.New("publish config to Nocos failed")
	}
	return nil
}

// GetConfigKeysByGroup will return all keys with the group
func (n *nacosDynamicConfiguration) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
	group = n.resolvedGroup(group)
	page, err := n.client.Client().SearchConfig(vo.SearchConfigParam{
		Search: "accurate",
		Group:  group,
		PageNo: 1,
		// actually it's impossible for user to create 9999 application under one group
		PageSize: maxKeysNum,
	})

	result := gxset.NewSet()
	if err != nil {
		return result, perrors.WithMessage(err, "can not find the configClient config")
	}
	for _, itm := range page.PageItems {
		result.Add(itm.DataId)
	}
	return result, nil
}

// GetRule Get router rule
func (n *nacosDynamicConfiguration) GetRule(key string, opts ...config_center.Option) (string, error) {
	tmpOpts := &config_center.Options{}
	for _, opt := range opts {
		opt(tmpOpts)
	}
	resolvedGroup := n.resolvedGroup(tmpOpts.Group)
	content, err := n.client.Client().GetConfig(vo.ConfigParam{
		DataId: key,
		Group:  resolvedGroup,
	})
	if err != nil {
		return "", perrors.WithStack(err)
	} else {
		return content, nil
	}
}

// Parser Get Parser
func (n *nacosDynamicConfiguration) Parser() parser.ConfigurationParser {
	return n.parser
}

// SetParser Set Parser
func (n *nacosDynamicConfiguration) SetParser(p parser.ConfigurationParser) {
	n.parser = p
}

// NacosClient Get Nacos Client
func (n *nacosDynamicConfiguration) NacosClient() *nacosClient.NacosConfigClient {
	return n.client
}

// SetNacosClient Set Nacos Client
func (n *nacosDynamicConfiguration) SetNacosClient(client *nacosClient.NacosConfigClient) {
	n.cltLock.Lock()
	n.client = client
	n.cltLock.Unlock()
}

// WaitGroup for wait group control, zk configClient listener & zk configClient container
func (n *nacosDynamicConfiguration) WaitGroup() *sync.WaitGroup {
	return &n.wg
}

// GetDone For nacos configClient control	RestartCallBack() bool
func (n *nacosDynamicConfiguration) GetDone() chan struct{} {
	return n.done
}

// GetURL Get URL
func (n *nacosDynamicConfiguration) GetURL() *common.URL {
	return n.url
}

// Destroy Destroy configuration instance
func (n *nacosDynamicConfiguration) Destroy() {
	close(n.done)
	n.wg.Wait()
	n.closeConfigs()
}

// resolvedGroup will regular the group. Now, it will replace the '/' with '-'.
// '/' is a special character for nacos
func (n *nacosDynamicConfiguration) resolvedGroup(group string) string {
	if len(group) <= 0 {
		group = n.url.GetParam(constant.NacosGroupKey, constant2.DEFAULT_GROUP)
		return group
	}
	return strings.ReplaceAll(group, "/", "-")
}

// IsAvailable Get available status
func (n *nacosDynamicConfiguration) IsAvailable() bool {
	select {
	case <-n.done:
		return false
	default:
		return true
	}
}

func (n *nacosDynamicConfiguration) closeConfigs() {
	// Close the old configClient first to close the tmp node
	n.client.Close()
	logger.Infof("begin to close provider n configClient")
}
