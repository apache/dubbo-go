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
	"sync"
	"time"
)

import (
	"github.com/apache/dubbo-getty"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
)

const (
	connDelay    = 3
	maxFailTimes = 15
)

type nacosClientFacade interface {
	NacosClient() *NacosClient
	SetNacosClient(*NacosClient)
	// WaitGroup for wait group control, zk client listener & zk client container
	WaitGroup() *sync.WaitGroup
	// GetDone For nacos client control	RestartCallBack() bool
	GetDone() chan struct{}
	common.Node
}

// HandleClientRestart Restart client handler
func HandleClientRestart(r nacosClientFacade) {
	var (
		err       error
		failTimes int
	)

	defer r.WaitGroup().Done()
LOOP:
	for {
		select {
		case <-r.GetDone():
			logger.Warnf("(NacosProviderRegistry)reconnectNacosRegistry goroutine exit now...")
			break LOOP
			// re-register all services
		case <-r.NacosClient().Done():
			r.NacosClient().Close()
			nacosName := r.NacosClient().name
			nacosAddress := r.NacosClient().NacosAddrs
			r.SetNacosClient(nil)

			// Connect nacos until success.
			failTimes = 0
			for {
				select {
				case <-r.GetDone():
					logger.Warnf("(NacosProviderRegistry)reconnectZkRegistry goroutine exit now...")
					break LOOP
				case <-getty.GetTimeWheel().After(time.Duration(failTimes*connDelay) * time.Second): // Prevent crazy reconnection nacos.
				}
				err = ValidateNacosClient(r, WithNacosName(nacosName))
				logger.Infof("NacosProviderRegistry.validateNacosClient(nacosAddr{%s}) = error{%#v}",
					nacosAddress, perrors.WithStack(err))
				if err == nil {
					break
				}
				failTimes++
				if maxFailTimes <= failTimes {
					failTimes = maxFailTimes
				}
			}
		}
	}
}
