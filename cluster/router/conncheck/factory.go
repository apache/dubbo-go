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
)

func init() {
	extension.SetRouterFactory(constant.ConnCheckRouterName, newConnCheckRouteFactory)
}

// ConnCheckRouteFactory is the factory to create conn check router, it aims at filter ip with unhealthy status
// the unhealthy status is storied in protocol/rpc_status.go with sync.Map
type ConnCheckRouteFactory struct {
}

// newConnCheckRouteFactory construct a new ConnCheckRouteFactory
func newConnCheckRouteFactory() router.PriorityRouterFactory {
	return &ConnCheckRouteFactory{}
}

// NewPriorityRouter construct a new NewConnCheckRouter via url
func (f *ConnCheckRouteFactory) NewPriorityRouter(url *common.URL, notify chan struct{}) (router.PriorityRouter, error) {
	return NewConnCheckRouter(url, notify)
}
