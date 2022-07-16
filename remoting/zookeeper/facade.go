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
	"time"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type ZkClientFacade interface {
	ZkClient() *gxzookeeper.ZookeeperClient
	SetZkClient(*gxzookeeper.ZookeeperClient)
	ZkClientLock() *sync.Mutex
	WaitGroup() *sync.WaitGroup // for wait group control, zk client listener & zk client container
	Done() chan struct{}        // for registry destroy
	RestartCallBack() bool
	GetURL() *common.URL
}

// HandleClientRestart keeps the connection between client and server
// This method should be used only once. You can use handleClientRestart() in package registry.
func HandleClientRestart(r ZkClientFacade) {
	defer r.WaitGroup().Done()
	for {
		select {
		case <-r.ZkClient().Reconnect():
			r.RestartCallBack()
			time.Sleep(10 * time.Microsecond)
		case <-r.Done():
			logger.Warnf("receive registry destroy event, quit client restart handler")
			return
		}
	}
}
