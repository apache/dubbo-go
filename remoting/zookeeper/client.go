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
	"time"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

const (
	// ConnDelay connection delay interval
	ConnDelay = 3
	// MaxFailTimes max fail times
	MaxFailTimes = 3
)

var (
	errNilZkClientConn = perrors.New("zookeeper client{conn} is nil")
	errNilChildren     = perrors.Errorf("has none children")
	errNilNode         = perrors.Errorf("node does not exist")
)

// ValidateZookeeperClient validates client and sets options
func ValidateZookeeperClient(container ZkClientFacade, zkName string) error {
	lock := container.ZkClientLock()
	url := container.GetUrl()

	lock.Lock()
	defer lock.Unlock()

	if container.ZkClient() == nil {
		// in dubbo, every registry only connect one node, so this is []string{r.Address}
		timeout, paramErr := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
		if paramErr != nil {
			logger.Errorf("timeout config %v is invalid, err is %v",
				url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT), paramErr.Error())
			return perrors.WithMessagef(paramErr, "newZookeeperClient(address:%+v)", url.Location)
		}
		zkAddresses := strings.Split(url.Location, ",")
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
