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

package directory

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

type staticDirectory struct {
	BaseDirectory
	invokers []protocol.Invoker
}

func NewStaticDirectory(invokers []protocol.Invoker) *staticDirectory {
	var url common.URL

	if len(invokers) > 0 {
		url = invokers[0].GetUrl()
	}
	return &staticDirectory{
		BaseDirectory: NewBaseDirectory(&url),
		invokers:      invokers,
	}
}

//for-loop invokers ,if all invokers is available ,then it means directory is available
func (dir *staticDirectory) IsAvailable() bool {
	if len(dir.invokers) == 0 {
		return false
	}
	for _, invoker := range dir.invokers {
		if !invoker.IsAvailable() {
			return false
		}
	}
	return true
}

func (dir *staticDirectory) List(invocation protocol.Invocation) []protocol.Invoker {
	//TODO:Here should add router
	return dir.invokers
}

func (dir *staticDirectory) Destroy() {
	dir.BaseDirectory.Destroy(func() {
		for _, ivk := range dir.invokers {
			ivk.Destroy()
		}
		dir.invokers = []protocol.Invoker{}
	})
}
