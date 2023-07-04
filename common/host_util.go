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

package common

import (
	"github.com/dubbogo/gost/log/logger"
	"os"
	"strconv"
)

import (
	gxnet "github.com/dubbogo/gost/net"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

var localIp string
var localHostname string

func GetLocalIp() string {
	if len(localIp) != 0 {
		return localIp
	}
	localIp, _ = gxnet.GetLocalIP()
	return localIp
}

func GetLocalHostName() string {
	if len(localHostname) != 0 {
		return localHostname
	}
	hostname, err := os.Hostname()
	if err != nil {
		logger.Errorf("can not get local hostname")
	}
	localHostname = hostname
	return localHostname
}

func HandleRegisterIPAndPort(url *URL) {
	// if developer define registry port and ip, use it first.
	if ipToRegistry := os.Getenv(constant.DubboIpToRegistryKey); len(ipToRegistry) > 0 {
		url.Ip = ipToRegistry
	}
	if len(url.Ip) == 0 {
		url.Ip = GetLocalIp()
	}
	if portToRegistry := os.Getenv(constant.DubboPortToRegistryKey); isValidPort(portToRegistry) {
		url.Port = portToRegistry
	}
	if len(url.Port) == 0 || url.Port == "0" {
		url.Port = constant.DubboDefaultPortToRegistry
	}
}

func isValidPort(port string) bool {
	if len(port) == 0 {
		return false
	}

	portInt, err := strconv.Atoi(port)
	return err == nil && portInt > 0 && portInt < 65536
}
