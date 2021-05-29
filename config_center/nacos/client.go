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
	"time"
)

import (
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/remoting/nacos"
)

// NacosConfigClient Nacos client
type NacosConfigClient struct {
	name       string
	NacosAddrs []string
	sync.Mutex // for Client
	client     *config_client.IConfigClient
	exit       chan struct{}
	Timeout    time.Duration
	once       sync.Once
	onceClose  func()
}

// Client Get NacosConfigClient
func (n *NacosConfigClient) Client() *config_client.IConfigClient {
	return n.client
}

// SetClient Set NacosConfigClient
func (n *NacosConfigClient) SetClient(client *config_client.IConfigClient) {
	n.Lock()
	n.client = client
	n.Unlock()
}

type option func(*options)

type options struct {
	nacosName string
	// client    *NacosConfigClient
}

// WithNacosName Set nacos name
func WithNacosName(name string) option {
	return func(opt *options) {
		opt.nacosName = name
	}
}

// ValidateNacosClient Validate nacos client , if null then create it
func ValidateNacosClient(container nacosClientFacade, opts ...option) error {
	if container == nil {
		return perrors.Errorf("container can not be null")
	}
	os := &options{}
	for _, opt := range opts {
		opt(os)
	}

	url := container.GetURL()
	timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
	if err != nil {
		logger.Errorf("invalid timeout config %+v,got err %+v",
			url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT), err)
		return perrors.WithMessagef(err, "newNacosClient(address:%+v)", url.Location)
	}
	nacosAddresses := strings.Split(url.Location, ",")
	if container.NacosClient() == nil {
		// in dubbo ,every registry only connect one node ,so this is []string{r.Address}
		newClient, err := newNacosClient(os.nacosName, nacosAddresses, timeout, url)
		if err != nil {
			logger.Errorf("newNacosClient(name{%s}, nacos address{%v}, timeout{%d}) = error{%v}",
				os.nacosName, url.Location, timeout.String(), err)
			return perrors.WithMessagef(err, "newNacosClient(address:%+v)", url.Location)
		}
		container.SetNacosClient(newClient)
	}

	if container.NacosClient().Client() == nil {
		configClient, err := nacos.NewNacosConfigClientByUrl(url)
		if err != nil {
			logger.Errorf("nacos.NewNacosConfigClientByUrl(url:%v) = err %+v", url, err)
			return perrors.WithMessagef(err, "newNacosClient(address:%+v)", url.Location)
		}
		container.NacosClient().SetClient(&configClient)
	}

	return perrors.WithMessagef(nil, "newNacosClient(address:%+v)", url.PrimitiveURL)
}

func newNacosClient(name string, nacosAddrs []string, timeout time.Duration, url *common.URL) (*NacosConfigClient, error) {
	var (
		err error
		n   *NacosConfigClient
	)

	n = &NacosConfigClient{
		name:       name,
		NacosAddrs: nacosAddrs,
		Timeout:    timeout,
		exit:       make(chan struct{}),
		onceClose: func() {
			close(n.exit)
		},
	}

	configClient, err := nacos.NewNacosConfigClientByUrl(url)
	if err != nil {
		logger.Errorf("initNacosConfigClient(addr:%+v,timeout:%v,url:%v) = err %+v",
			nacosAddrs, timeout.String(), url, err)
		return n, perrors.WithMessagef(err, "newNacosClient(address:%+v)", url.Location)
	}
	n.SetClient(&configClient)

	return n, nil
}

// Done Get nacos client exit signal
func (n *NacosConfigClient) Done() <-chan struct{} {
	return n.exit
}

func (n *NacosConfigClient) stop() bool {
	select {
	case <-n.exit:
		return true
	default:
		n.once.Do(n.onceClose)
	}

	return false
}

// NacosClientValid Get nacos client valid status
func (n *NacosConfigClient) NacosClientValid() bool {
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

// Close Close nacos client , then set null
func (n *NacosConfigClient) Close() {
	if n == nil {
		return
	}

	n.stop()
	n.SetClient(nil)
}
