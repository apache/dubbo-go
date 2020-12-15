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
	"github.com/apache/dubbo-getty"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
)

type ZkClientFacade interface {
	ZkClient() *ZookeeperClient
	SetZkClient(*ZookeeperClient)
	ZkClientLock() *sync.Mutex
	WaitGroup() *sync.WaitGroup // for wait group control, zk client listener & zk client container
	Done() chan struct{}        // for zk client control
	RestartCallBack() bool
	GetUrl() *common.URL
}

// HandleClientRestart keeps the connection between client and server
func HandleClientRestart(r ZkClientFacade) {
	var (
		err error

		failTimes int
	)

LOOP:
	for {
		select {
		case <-r.Done():
			r.WaitGroup().Done() // dec the wg when registry is closed
			logger.Warnf("(ZkProviderRegistry)reconnectZkRegistry goroutine exit now...")
			break LOOP
			// re-register all services
		case <-r.ZkClient().Done():
			r.ZkClientLock().Lock()
			r.ZkClient().Close()
			zkName := r.ZkClient().name
			zkAddress := r.ZkClient().ZkAddrs
			r.SetZkClient(nil)
			r.ZkClientLock().Unlock()
			r.WaitGroup().Done() // dec the wg when zk client is closed

			// Connect zk until success.
			failTimes = 0
			for {
				select {
				case <-r.Done():
					r.WaitGroup().Done() // dec the wg when registry is closed
					logger.Warnf("(ZkProviderRegistry)reconnectZkRegistry goroutine exit now...")
					break LOOP
				case <-getty.GetTimeWheel().After(timeSecondDuration(failTimes * ConnDelay)): // Prevent crazy reconnection zk.
				}
				err = ValidateZookeeperClient(r, WithZkName(zkName))
				logger.Infof("ZkProviderRegistry.validateZookeeperClient(zkAddr{%s}) = error{%#v}",
					zkAddress, perrors.WithStack(err))
				if err == nil && r.RestartCallBack() {
					break
				}
				failTimes++
				if MaxFailTimes <= failTimes {
					failTimes = MaxFailTimes
				}
			}
		}
	}
}
