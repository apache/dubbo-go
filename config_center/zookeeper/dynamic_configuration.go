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
	"sync"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/remoting"
	"github.com/apache/dubbo-go/remoting/zookeeper"
)

const ZK_CLIENT = "zk config_center"

type ZookeeperDynamicConfiguration struct {
	url      common.URL
	rootPath string
	wg       sync.WaitGroup
	cltLock  sync.Mutex
	done     chan struct{}
	client   *zookeeper.ZookeeperClient

	listenerLock sync.Mutex
	listener     *zookeeper.ZkEventListener
}

func NewZookeeperDynamicConfiguration(url common.URL) (config_center.DynamicConfiguration, error) {
	c := &ZookeeperDynamicConfiguration{
		url:      url,
		rootPath: "/" + url.GetParam(constant.CONFIG_NAMESPACE_KEY, config_center.DEFAULT_GROUP) + "/config",
	}
	err := zookeeper.ValidateZookeeperClient(c, zookeeper.WithZkName(ZK_CLIENT))
	if err != nil {
		return nil, err
	}
	c.wg.Add(1)
	go zookeeper.HandleClientRestart(c)

	c.listener = zookeeper.NewZkEventListener(c.client)
	//c.configListener = NewRegistryConfigurationListener(c.client, c)
	//c.dataListener = NewRegistryDataListener(c.configListener)
	return c, nil

}

func (*ZookeeperDynamicConfiguration) AddListener(key string, listener remoting.ConfigurationListener, opions ...config_center.Option) {

}

func (*ZookeeperDynamicConfiguration) RemoveListener(key string, listener remoting.ConfigurationListener, opions ...config_center.Option) {

}

func (*ZookeeperDynamicConfiguration) GetConfig(key string, opions ...config_center.Option) string {
	return ""
}

func (*ZookeeperDynamicConfiguration) GetConfigs(key string, opions ...config_center.Option) string {
	return ""
}

func (r *ZookeeperDynamicConfiguration) ZkClient() *zookeeper.ZookeeperClient {
	return r.client
}

func (r *ZookeeperDynamicConfiguration) SetZkClient(client *zookeeper.ZookeeperClient) {
	r.client = client
}

func (r *ZookeeperDynamicConfiguration) ZkClientLock() *sync.Mutex {
	return &r.cltLock
}

func (r *ZookeeperDynamicConfiguration) WaitGroup() *sync.WaitGroup {
	return &r.wg
}

func (r *ZookeeperDynamicConfiguration) GetDone() chan struct{} {
	return r.done
}

func (r *ZookeeperDynamicConfiguration) GetUrl() common.URL {
	return r.url
}

func (r *ZookeeperDynamicConfiguration) Destroy() {
	if r.listener != nil {
		r.listener.Close()
	}
	close(r.done)
	r.wg.Wait()
	r.closeConfigs()
}

func (r *ZookeeperDynamicConfiguration) IsAvailable() bool {
	select {
	case <-r.done:
		return false
	default:
		return true
	}
}

func (r *ZookeeperDynamicConfiguration) closeConfigs() {
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	logger.Infof("begin to close provider zk client")
	// 先关闭旧client，以关闭tmp node
	r.client.Close()
	r.client = nil
}

func (r *ZookeeperDynamicConfiguration) RestartCallBack() bool {
	return true
}
