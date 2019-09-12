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

package utils

import (
	"net"
)

import (
	"github.com/apache/dubbo-go/common/logger"
)

var localIp = ""

func init() {
	initLocalIp()
}

func initLocalIp() {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Error("[utils][initLocalIp] error", err)
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				localIp = ipNet.IP.String()
			}
		}
	}

	if len(localIp) == 0 {
		logger.Error("[utils][initLocalIp] can not get local IP")
	}
}

func GetLocalIP() string {
	return localIp
}
