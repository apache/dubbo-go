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
	"github.com/dubbogo/getty"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
)

type zkClientFacade interface {
	ZkClient() *ZookeeperClient
	SetZkClient(*ZookeeperClient)
	ZkClientLock() *sync.Mutex
	WaitGroup() *sync.WaitGroup //for wait group control, zk client listener & zk client container
	GetDone() chan struct{}     //for zk client control
	RestartCallBack() bool
	common.Node
}

func HandleClientRestart(r zkClientFacade) {
	var (
		err error

		failTimes int
	)

	defer r.WaitGroup().Done()
LOOP:
	for {
		select {
		case <-r.GetDone():
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

			// Connect zk until success.
			failTimes = 0
			for {
				select {
				case <-r.GetDone():
					logger.Warnf("(ZkProviderRegistry)reconnectZkRegistry goroutine exit now...")
					break LOOP
				case <-getty.GetTimeWheel().After(timeSecondDuration(failTimes * ConnDelay)): // Prevent crazy reconnection zk.
				}
				err = ValidateZookeeperClient(r, WithZkName(zkName))
				logger.Infof("ZkProviderRegistry.validateZookeeperClient(zkAddr{%s}) = error{%#v}",
					zkAddress, perrors.WithStack(err))
				if err == nil {
					if r.RestartCallBack() {
						break
					}
				}
				failTimes++
				if MaxFailTimes <= failTimes {
					failTimes = MaxFailTimes
				}
			}
		}
	}
}
