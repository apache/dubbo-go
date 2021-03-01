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

package conncheck

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
)

func init() {
	extension.SetConnChecker(constant.DEFAULT_CONN_CHECKER, NewDefaultConnChecker)
}

// DefaultConnChecker is the default implementation of ConnChecker, which determines the health status of invoker conn
type DefaultConnChecker struct {
}

// IsConnHealthy evaluates the healthy state on the given Invoker based on the number of successive bad request
// and the current active request
func (c *DefaultConnChecker) IsConnHealthy(invoker protocol.Invoker) bool {
	return protocol.GetInvokerHealthyStatus(invoker)
}

// NewDefaultConnChecker constructs a new DefaultConnChecker based on the url
func NewDefaultConnChecker(url *common.URL) router.ConnChecker {
	return &DefaultConnChecker{}
}
