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

package base

import (
	"sync"
)

import (
	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// Directory Abstract implementation of Directory: Invoker list returned from this Directory's list method have been filtered by Routers
type Directory struct {
	url       *common.URL
	destroyed *atomic.Bool
	// this mutex for change the properties in BaseDirectory, like routerChain , destroyed etc
	mutex       sync.Mutex
	routerChain router.Chain
}

// NewDirectory Create BaseDirectory with URL
func NewDirectory(url *common.URL) Directory {
	return Directory{
		url:         url,
		destroyed:   atomic.NewBool(false),
		routerChain: &chain.RouterChain{},
	}
}

// RouterChain Return router chain in directory
func (dir *Directory) RouterChain() router.Chain {
	return dir.routerChain
}

// SetRouterChain Set router chain in directory
func (dir *Directory) SetRouterChain(routerChain router.Chain) {
	dir.mutex.Lock()
	defer dir.mutex.Unlock()
	dir.routerChain = routerChain
}

// GetURL Get URL
func (dir *Directory) GetURL() *common.URL {
	return dir.url
}

// GetDirectoryUrl Get URL instance
func (dir *Directory) GetDirectoryUrl() *common.URL {
	return dir.url
}

func (dir *Directory) isProperRouter(url *common.URL) bool {
	app := url.GetParam(constant.ApplicationKey, "")
	dirApp := dir.GetURL().GetParam(constant.ApplicationKey, "")
	if len(dirApp) == 0 && dir.GetURL().SubURL != nil {
		dirApp = dir.GetURL().SubURL.GetParam(constant.ApplicationKey, "")
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
func (dir *Directory) Destroy(doDestroy func()) {
	if dir.destroyed.CAS(false, true) {
		dir.mutex.Lock()
		doDestroy()
		dir.mutex.Unlock()
	}
}

// IsAvailable Once directory init finish, it will change to true
func (dir *Directory) IsAvailable() bool {
	return !dir.destroyed.Load()
}
