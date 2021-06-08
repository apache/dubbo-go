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

package directory

import (
	"sync"
)

import (
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/cluster/router/chain"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
)

// BaseDirectory Abstract implementation of Directory: Invoker list returned from this Directory's list method have been filtered by Routers
type BaseDirectory struct {
	url       *common.URL
	destroyed *atomic.Bool
	// this mutex for change the properties in BaseDirectory, like routerChain , destroyed etc
	mutex       sync.Mutex
	routerChain router.Chain
}

// NewBaseDirectory Create BaseDirectory with URL
func NewBaseDirectory(url *common.URL) BaseDirectory {
	dir := BaseDirectory{
		url:         url,
		destroyed:   atomic.NewBool(false),
		routerChain: &chain.RouterChain{},
	}
	// start to listen notify
	dir.routerChain.Loop()
	return dir
}

// RouterChain Return router chain in directory
func (dir *BaseDirectory) RouterChain() router.Chain {
	return dir.routerChain
}

// SetRouterChain Set router chain in directory
func (dir *BaseDirectory) SetRouterChain(routerChain router.Chain) {
	dir.mutex.Lock()
	defer dir.mutex.Unlock()
	dir.routerChain = routerChain
}

// GetURL Get URL
func (dir *BaseDirectory) GetURL() *common.URL {
	return dir.url
}

// GetDirectoryUrl Get URL instance
func (dir *BaseDirectory) GetDirectoryUrl() *common.URL {
	return dir.url
}

// AddRouters Convert url to routers and add them into dir.routerChain
func (dir *BaseDirectory) AddRouters(urls []*common.URL) {
	if len(urls) == 0 {
		return
	}

	routers := make([]router.PriorityRouter, 0, len(urls))
	for _, url := range urls {
		if r := dir.buildRouter(url); r != nil {
			routers = append(routers, r)
		}
	}

	logger.Infof("Init file condition router success, size: %v", len(routers))
	dir.mutex.Lock()
	dir.routerChain.AddRouters(routers)
	dir.mutex.Unlock()
}

func (dir *BaseDirectory) buildRouter(url *common.URL) router.PriorityRouter {
	routerKey := url.GetParam(constant.ROUTER_KEY, "")
	if len(routerKey) == 0 {
		return nil
	}

	if url.Protocol == constant.CONDITION_ROUTE_PROTOCOL {
		if !dir.isProperRouter(url) {
			return nil
		}
	}

	factory := extension.GetRouterFactory(url.Protocol)
	r, err := factory.NewPriorityRouter(url, dir.routerChain.GetNotifyChan())
	if err != nil {
		logger.Errorf("Create router fail. router key: %s, url:%s, error: %+v", routerKey, url.Service(), err)
		return nil
	}
	return r
}

func (dir *BaseDirectory) isProperRouter(url *common.URL) bool {
	app := url.GetParam(constant.APPLICATION_KEY, "")
	dirApp := dir.GetURL().GetParam(constant.APPLICATION_KEY, "")
	if len(dirApp) == 0 && dir.GetURL().SubURL != nil {
		dirApp = dir.GetURL().SubURL.GetParam(constant.APPLICATION_KEY, "")
	}
	serviceKey := dir.GetURL().ServiceKey()
	if len(serviceKey) == 0 {
		serviceKey = dir.GetURL().SubURL.ServiceKey()
	}
	if len(app) > 0 && app == dirApp {
		return true
	}
	if url.ServiceKey() == serviceKey {
		return true
	}
	return false
}

// Destroy Destroy
func (dir *BaseDirectory) Destroy(doDestroy func()) {
	if dir.destroyed.CAS(false, true) {
		dir.mutex.Lock()
		doDestroy()
		dir.mutex.Unlock()
	}
}

// IsAvailable Once directory init finish, it will change to true
func (dir *BaseDirectory) IsAvailable() bool {
	return !dir.destroyed.Load()
}
