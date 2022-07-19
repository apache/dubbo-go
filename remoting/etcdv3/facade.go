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

package etcdv3

import (
	"sync"
	"time"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type clientFacade interface {
	Client() *gxetcd.Client
	SetClient(client *gxetcd.Client)
	ClientLock() *sync.Mutex
	WaitGroup() *sync.WaitGroup // for wait group control, etcd client listener & etcd client container
	Done() chan struct{}        // for etcd client control
	RestartCallBack() bool
	common.Node
}

// HandleClientRestart keeps the connection between client and server
// This method should be used only once. You can use handleClientRestart() in package registry.
func HandleClientRestart(r clientFacade) {
	defer r.WaitGroup().Done()
	for {
		select {
		case <-r.Client().GetCtx().Done():
			r.RestartCallBack()
			// re-register all services
			time.Sleep(10 * time.Microsecond)
		case <-r.Done():
			logger.Warnf("(ETCDV3ProviderRegistry)reconnectETCDV3 goroutine exit now...")
			return
		}
	}
}
