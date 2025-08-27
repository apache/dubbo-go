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

	dir.RouterChain().SetInvokers(invokers)
	return dir
}

// for-loop invokers ,if all invokers is available ,then it means directory is available
func (dir *directory) IsAvailable() bool {
	if dir.IsDestroyed() {
		return false
	}

	if len(dir.invokers) == 0 {
		return false
	}
	for _, invoker := range dir.invokers {
		if !invoker.IsAvailable() {
			return false
		}
	}
	return true
}

// List List invokers
func (dir *directory) List(invocation protocolbase.Invocation) []protocolbase.Invoker {
	l := len(dir.invokers)
	invokers := make([]protocolbase.Invoker, l)
	copy(invokers, dir.invokers)
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
