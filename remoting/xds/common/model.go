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
	"net"
	"strings"
)

type Addr struct {
	HostnameOrIP string
	Port         string
}

func NewAddr(addr string) Addr {
	host, port, _ := net.SplitHostPort(addr)
	return Addr{
		HostnameOrIP: host,
		Port:         port,
	}
}

func (a *Addr) String() string {
	return net.JoinHostPort(a.HostnameOrIP, a.Port)
}

type Cluster struct {
	Bound  string
	Addr   Addr
	Subset string
}

func NewCluster(clusterID string) Cluster {
	clusterIDs := strings.Split(clusterID, "|")
	return Cluster{
		Bound: clusterIDs[0],
		Addr: Addr{
			Port:         clusterIDs[1],
			HostnameOrIP: clusterIDs[3],
		},
		Subset: clusterIDs[2],
	}
}
