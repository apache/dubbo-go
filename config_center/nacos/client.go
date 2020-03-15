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
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	nacosconst "github.com/nacos-group/nacos-sdk-go/common/constant"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

const logDir = "logs/nacos/log"

// NacosClient Nacos client
type NacosClient struct {
	name       string
	NacosAddrs []string
	sync.Mutex // for Client
	client     *config_client.IConfigClient
	exit       chan struct{}
	Timeout    time.Duration
	once       sync.Once
	onceClose  func()
}

// Client Get Client
func (n *NacosClient) Client() *config_client.IConfigClient {
	return n.client
}

// SetClient Set client
func (n *NacosClient) SetClient(client *config_client.IConfigClient) {
	n.Lock()
	n.client = client
	n.Unlock()
}

type option func(*options)

type options struct {
	nacosName string
	client    *NacosClient
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

	url := container.GetUrl()

	if container.NacosClient() == nil {
		//in dubbo ,every registry only connect one node ,so this is []string{r.Address}
		timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
		if err != nil {
			logger.Errorf("timeout config %v is invalid ,err is %v",
				url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT), err.Error())
			return perrors.WithMessagef(err, "newNacosClient(address:%+v)", url.Location)
		}
		nacosAddresses := strings.Split(url.Location, ",")
		newClient, err := newNacosClient(os.nacosName, nacosAddresses, timeout)
		if err != nil {
			logger.Warnf("newNacosClient(name{%s}, nacos address{%v}, timeout{%d}) = error{%v}",
				os.nacosName, url.Location, timeout.String(), err)
			return perrors.WithMessagef(err, "newNacosClient(address:%+v)", url.Location)
		}
		container.SetNacosClient(newClient)
	}

	if container.NacosClient().Client() == nil {
		svrConfList := []nacosconst.ServerConfig{}
		for _, nacosAddr := range container.NacosClient().NacosAddrs {
			split := strings.Split(nacosAddr, ":")
			port, err := strconv.ParseUint(split[1], 10, 64)
			if err != nil {
				logger.Warnf("nacos addr port parse error ,error message is %v", err)
				continue
			}
			svrconf := nacosconst.ServerConfig{
				IpAddr: split[0],
				Port:   port,
			}
			svrConfList = append(svrConfList, svrconf)
		}

		client, err := clients.CreateConfigClient(map[string]interface{}{
			"serverConfigs": svrConfList,
			"clientConfig": nacosconst.ClientConfig{
				TimeoutMs:           uint64(int32(container.NacosClient().Timeout / time.Millisecond)),
				ListenInterval:      10000,
				NotLoadCacheAtStart: true,
				LogDir:              logDir,
			},
		})

		container.NacosClient().SetClient(&client)
		if err != nil {
			logger.Errorf("nacos create config client error:%v", err)
		}
	}

	return perrors.WithMessagef(nil, "newNacosClient(address:%+v)", url.PrimitiveURL)
}

func newNacosClient(name string, nacosAddrs []string, timeout time.Duration) (*NacosClient, error) {
	var (
		err error
		n   *NacosClient
	)

	n = &NacosClient{
		name:       name,
		NacosAddrs: nacosAddrs,
		Timeout:    timeout,
		exit:       make(chan struct{}),
		onceClose: func() {
			close(n.exit)
		},
	}

	svrConfList := make([]nacosconst.ServerConfig, 0, len(n.NacosAddrs))
	for _, nacosAddr := range n.NacosAddrs {
		split := strings.Split(nacosAddr, ":")
		port, err := strconv.ParseUint(split[1], 10, 64)
		if err != nil {
			logger.Warnf("convert port , source:%s , error:%v ", split[1], err)
			continue
		}
		svrconf := nacosconst.ServerConfig{
			IpAddr: split[0],
			Port:   port,
		}
		svrConfList = append(svrConfList, svrconf)
	}
	client, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": svrConfList,
		"clientConfig": nacosconst.ClientConfig{
			TimeoutMs:           uint64(timeout / time.Millisecond),
			ListenInterval:      20000,
			NotLoadCacheAtStart: true,
			LogDir:              logDir,
		},
	})
	n.SetClient(&client)
	if err != nil {
		return nil, perrors.WithMessagef(err, "nacos clients.CreateConfigClient(nacosAddrs:%+v)", nacosAddrs)
	}

	return n, nil
}

// Done Get nacos client exit signal
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

// NacosClientValid Get nacos client valid status
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

// Close Close nacos client , then set null
func (n *NacosClient) Close() {
	if n == nil {
		return
	}

	n.stop()
	n.SetClient(nil)
	logger.Warnf("nacosClient{name:%s, nacos addr:%s} exit now.", n.name, n.NacosAddrs)
}
