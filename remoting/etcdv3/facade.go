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
	"github.com/apache/dubbo-getty"
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
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
func HandleClientRestart(r clientFacade) {
	var (
		err       error
		failTimes int
	)

	defer r.WaitGroup().Done()
LOOP:
	for {
		select {
		case <-r.Done():
			logger.Warnf("(ETCDV3ProviderRegistry)reconnectETCDV3 goroutine exit now...")
			break LOOP
			// re-register all services
		case <-r.Client().Done():
			r.ClientLock().Lock()
			clientName := gxetcd.RegistryETCDV3Client
			timeout, _ := time.ParseDuration(r.GetURL().GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
			endpoints := r.Client().GetEndPoints()
			r.Client().Close()
			r.SetClient(nil)
			r.ClientLock().Unlock()

			// try to connect to etcd,
			failTimes = 0
			for {
				select {
				case <-r.Done():
					logger.Warnf("(ETCDV3ProviderRegistry)reconnectETCDRegistry goroutine exit now...")
					break LOOP
				case <-getty.GetTimeWheel().After(timeSecondDuration(failTimes * gxetcd.ConnDelay)): // avoid connect frequent
				}
				err = ValidateClient(
					r,
					gxetcd.WithName(clientName),
					gxetcd.WithEndpoints(endpoints...),
					gxetcd.WithTimeout(timeout),
					gxetcd.WithHeartbeat(1),
				)
				logger.Infof("ETCDV3ProviderRegistry.validateETCDV3Client(etcd Addr{%s}) = error{%#v}",
					endpoints, perrors.WithStack(err))
				if err == nil && r.RestartCallBack() {
					break
				}
				failTimes++
				if gxetcd.MaxFailTimes <= failTimes {
					failTimes = gxetcd.MaxFailTimes
				}
			}
		}
	}
}
