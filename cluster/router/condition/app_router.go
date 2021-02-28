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

package condition

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

const (
	// Default priority for application router
	appRouterDefaultPriority = int64(150)
)

// AppRouter For listen application router with config center
type AppRouter struct {
	listenableRouter
	notify interface{}
}

// NewAppRouter Init AppRouter by url
func NewAppRouter(url *common.URL, notify chan struct{}) (*AppRouter, error) {
	if url == nil {
		return nil, perrors.Errorf("No route URL for create app router!")
	}
	appRouter, err := newListenableRouter(url, url.GetParam(constant.APPLICATION_KEY, ""), notify)
	if err != nil {
		return nil, err
	}
	appRouter.priority = appRouterDefaultPriority
	return appRouter, nil
}
