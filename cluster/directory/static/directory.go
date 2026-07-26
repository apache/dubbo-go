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

package static

import (
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	"dubbo.apache.org/dubbo-go/v3/common"
	protocolbase "dubbo.apache.org/dubbo-go/v3/protocol/base"
)

type directory struct {
	*base.Directory
	invokers []protocolbase.Invoker
	// lock guards invokers against the race between List/IsAvailable (read) and
	// Destroy (write). base.Directory.mutex is unexported and not accessible
	// from this package. See #3520.
	lock sync.RWMutex
}

// NewDirectory Create a new staticDirectory with invokers
func NewDirectory(invokers []protocolbase.Invoker) *directory {
	var url *common.URL

	if len(invokers) > 0 {
		url = invokers[0].GetURL()
	}
	dir := &directory{
		Directory: base.NewDirectory(url),
		invokers:  invokers,
	}

	err := dir.BuildRouterChain(invokers, url)
	if err != nil {
		logger.Errorf("[Cluster][Directory] build router chain failed, err=%v", err)
		dir.RouterChain().SetInvokers(invokers)
	}

	return dir
}

// for-loop invokers ,if all invokers is available ,then it means directory is available
func (dir *directory) IsAvailable() bool {
	if dir.IsDestroyed() {
		return false
	}

	// Snapshot invokers under lock: Destroy reassigns the slice header (under
	// lock below), so an unlocked read races it. See #3520.
	dir.lock.RLock()
	invokers := dir.invokers
	dir.lock.RUnlock()

	if len(invokers) == 0 {
		return false
	}
	for _, invoker := range invokers {
		if !invoker.IsAvailable() {
			return false
		}
	}
	return true
}

// List List invokers
func (dir *directory) List(invocation protocolbase.Invocation) []protocolbase.Invoker {
	// Snapshot invokers under lock, then release before touching the router
	// chain (which takes base.Directory.mutex) so the two locks never nest.
	dir.lock.RLock()
	l := len(dir.invokers)
	invokers := make([]protocolbase.Invoker, l)
	copy(invokers, dir.invokers)
	dir.lock.RUnlock()

	routerChain := dir.RouterChain()

	if routerChain == nil {
		return invokers
	}
	dirUrl := dir.GetURL()
	return routerChain.Route(dirUrl, invocation)
}

// Destroy Destroy
func (dir *directory) Destroy() {
	dir.DoDestroy(func() {
		// Guard the invokers write with the same lock List/IsAvailable read
		// under. DoDestroy already holds base.Directory.mutex; this lock is
		// never held while waiting for base.Directory.mutex (List/IsAvailable
		// release it before calling RouterChain), so there is no deadlock.
		dir.lock.Lock()
		defer dir.lock.Unlock()
		for _, ivk := range dir.invokers {
			ivk.Destroy()
		}
		dir.invokers = []protocolbase.Invoker{}
	})
}

// BuildRouterChain build router chain by invokers
func (dir *directory) BuildRouterChain(invokers []protocolbase.Invoker, url *common.URL) error {
	if len(invokers) == 0 {
		return perrors.Errorf("invokers == null")
	}
	routerChain, e := chain.NewRouterChain(url)
	if e != nil {
		return e
	}
	routerChain.SetInvokers(dir.invokers)
	dir.SetRouterChain(routerChain)
	return nil
}

func (dir *directory) Subscribe(url *common.URL) error {
	panic("Static directory does not support subscribing to registry.")
}
