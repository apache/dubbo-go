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
)

import (
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type nacosClientFacade interface {
	NacosClient() *nacosClient.NacosConfigClient
	SetNacosClient(*nacosClient.NacosConfigClient)
	// WaitGroup for wait group control, zk configClient listener & zk configClient container
	WaitGroup() *sync.WaitGroup
	// GetDone For nacos configClient control	RestartCallBack() bool
	GetDone() chan struct{}
	common.Node
}

// HandleClientRestart Restart configClient handler
func HandleClientRestart(r nacosClientFacade) {
	defer r.WaitGroup().Done()
	for {
		select {
		case <-r.GetDone():
			logger.Warnf("(NacosProviderRegistry)reconnectNacosRegistry goroutine exit now...")
			return
		}
	}
}
