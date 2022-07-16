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
	"time"
)

import (
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/remoting/nacos"
)

// NacosClient Nacos configClient
type NacosClient struct {
	name         string
	NacosAddrs   []string
	sync.Mutex   // for Client
	configClient *nacosClient.NacosConfigClient
	exit         chan struct{}
	Timeout      time.Duration
	once         sync.Once
	onceClose    func()
}

// Client Get Client
func (n *NacosClient) Client() *nacosClient.NacosConfigClient {
	return n.configClient
}

// SetClient Set configClient
func (n *NacosClient) SetClient(configClient *nacosClient.NacosConfigClient) {
	n.Lock()
	n.configClient = configClient
	n.Unlock()
}

// ValidateNacosClient Validate nacos configClient , if null then create it
func ValidateNacosClient(container nacosClientFacade) error {
	if container == nil {
		return perrors.Errorf("container can not be null")
	}
	url := container.GetURL()
	if container.NacosClient() == nil || container.NacosClient().Client() == nil {
		// in dubbo ,every registry only connect one node ,so this is []string{r.Address}
		newClient, err := nacos.NewNacosConfigClientByUrl(url)
		if err != nil {
			logger.Errorf("ValidateNacosClient(name{%s}, nacos address{%v} = error{%v}", url.Location, err)
			return perrors.WithMessagef(err, "newNacosClient(address:%+v)", url.Location)
		}
		container.SetNacosClient(newClient)
	}
	return perrors.WithMessagef(nil, "newNacosClient(address:%+v)", url.PrimitiveURL)
}

// Done Get nacos configClient exit signal
func (n *NacosClient) Done() <-chan struct{} {
	return n.exit
}

func (n *NacosClient) stop() bool {
	select {
	case <-n.exit:
		return true
	default:
		n.once.Do(n.onceClose)
	}

	return false
}

// NacosClientValid Get nacos configClient valid status
func (n *NacosClient) NacosClientValid() bool {
	select {
	case <-n.exit:
		return false
	default:
	}

	valid := true
	n.Lock()
	if n.Client() == nil {
		valid = false
	}
	n.Unlock()

	return valid
}

// Close Close nacos configClient , then set null
func (n *NacosClient) Close() {
	if n == nil {
		return
	}

	n.stop()
	n.SetClient(nil)
}
