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
	"strings"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

const (
	ConnDelay    = 3 // connection delay interval
	MaxFailTimes = 3 // max fail times
)

// ValidateZookeeperClient validates client and sets options
func ValidateZookeeperClient(container ZkClientFacade, zkName string) error {
	lock := container.ZkClientLock()
	url := container.GetURL()

	lock.Lock()
	defer lock.Unlock()

	if container.ZkClient() == nil {
		// in dubbo, every registry only connect one node, so this is []string{r.Address}
		timeout := url.GetParamDuration(constant.ConfigTimeoutKey, constant.DefaultRegTimeout)

		zkAddresses := strings.Split(url.Location, ",")
		logger.Infof("[Zookeeper Client] New zookeeper client with name = %s, zkAddress = %s, timeout = %s", zkName, url.Location, timeout.String())
		newClient, cltErr := gxzookeeper.NewZookeeperClient(zkName, zkAddresses, true, gxzookeeper.WithZkTimeOut(timeout))
		if cltErr != nil {
			logger.Warnf("newZookeeperClient(name{%s}, zk address{%v}, timeout{%d}) = error{%v}",
				zkName, url.Location, timeout.String(), cltErr)
			return perrors.WithMessagef(cltErr, "newZookeeperClient(address:%+v)", url.Location)
		}
		container.SetZkClient(newClient)
	}
	return nil
}
