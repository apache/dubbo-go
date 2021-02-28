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
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
)

func init() {
	extension.SetRouterFactory(constant.ConditionRouterName, newConditionRouterFactory)
	extension.SetRouterFactory(constant.ConditionAppRouterName, newAppRouterFactory)
}

// ConditionRouterFactory Condition router factory
type ConditionRouterFactory struct{}

func newConditionRouterFactory() router.PriorityRouterFactory {
	return &ConditionRouterFactory{}
}

// NewPriorityRouter creates ConditionRouterFactory by URL
func (c *ConditionRouterFactory) NewPriorityRouter(url *common.URL, notify chan struct{}) (router.PriorityRouter, error) {
	return NewConditionRouter(url, notify)
}

// NewRouter Create FileRouterFactory by Content
func (c *ConditionRouterFactory) NewFileRouter(content []byte) (router.PriorityRouter, error) {
	return NewFileConditionRouter(content)
}

// AppRouterFactory Application router factory
type AppRouterFactory struct{}

func newAppRouterFactory() router.PriorityRouterFactory {
	return &AppRouterFactory{}
}

// NewPriorityRouter creates AppRouterFactory by URL
func (c *AppRouterFactory) NewPriorityRouter(url *common.URL, notify chan struct{}) (router.PriorityRouter, error) {
	return NewAppRouter(url, notify)
}
