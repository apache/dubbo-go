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

const NacosClientName = "nacos config_center"

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
		rootPath:     "/" + url.GetParam(constant.CONFIG_NAMESPACE_KEY, config_center.DEFAULT_GROUP) + "/config",
		url:          url,
		keyListeners: sync.Map{},
	}
	err := ValidateNacosClient(c, WithNacosName(NacosClientName))
	if err != nil {
		logger.Errorf("nacos client start error ,error message is %v", err)
		return nil, err
	}
	c.wg.Add(1)
	go HandleClientRestart(c)
	return c, err

}

func (n *nacosDynamicConfiguration) AddListener(key string, listener config_center.ConfigurationListener, opions ...config_center.Option) {
	n.addListener(key, listener)
}

func (n *nacosDynamicConfiguration) RemoveListener(key string, listener config_center.ConfigurationListener, opions ...config_center.Option) {
	n.removeListener(key, listener)
}

// 在nacos group是dubbo DataId是key configfile 或  appconfigfile
func (n *nacosDynamicConfiguration) GetProperties(key string, opts ...config_center.Option) (string, error) {

	tmpOpts := &config_center.Options{}
	for _, opt := range opts {
		opt(tmpOpts)
	}
	content, err := (*n.client.Client).GetConfig(vo.ConfigParam{
		DataId: key,
		Group:  tmpOpts.Group,
	})
	if err != nil {
		return "", perrors.WithStack(err)
	} else {
		return string(content), nil
	}

}

func (n *nacosDynamicConfiguration) GetInternalProperty(key string, opts ...config_center.Option) (string, error) {
	return n.GetProperties(key, opts...)
}

func (n *nacosDynamicConfiguration) GetRule(key string, opts ...config_center.Option) (string, error) {
	return n.GetProperties(key, opts...)
}

func (n *nacosDynamicConfiguration) Parser() parser.ConfigurationParser {
	return n.parser
}

func (n *nacosDynamicConfiguration) SetParser(p parser.ConfigurationParser) {
	n.parser = p
}

func (n *nacosDynamicConfiguration) NacosClient() *NacosClient {
	return n.client
}

func (n *nacosDynamicConfiguration) SetNacosClient(client *NacosClient) {
	n.client = client
}

func (n *nacosDynamicConfiguration) NacosClientLock() *sync.Mutex {
	return &n.cltLock
}

func (n *nacosDynamicConfiguration) WaitGroup() *sync.WaitGroup {
	return &n.wg
}

func (n *nacosDynamicConfiguration) GetDone() chan struct{} {
	return n.done
}

func (n *nacosDynamicConfiguration) GetUrl() common.URL {
	return *n.url
}

func (n *nacosDynamicConfiguration) Destroy() {
	close(n.done)
	n.wg.Wait()
	n.closeConfigs()
}

func (n *nacosDynamicConfiguration) IsAvailable() bool {
	select {
	case <-n.done:
		return false
	default:
		return true
	}
}

func (r *nacosDynamicConfiguration) closeConfigs() {
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	logger.Infof("begin to close provider zk client")
	// Close the old client first to close the tmp node
	r.client.Close()
	r.client = nil
}

func (r *nacosDynamicConfiguration) RestartCallBack() bool {
	return true
}
