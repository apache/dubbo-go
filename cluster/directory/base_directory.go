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
	"sync"
)
import (
	"go.uber.org/atomic"
)
import (
	"github.com/apache/dubbo-go/common"
)

type BaseDirectory struct {
	url       *common.URL
	destroyed *atomic.Bool
	mutex     sync.Mutex
}

func NewBaseDirectory(url *common.URL) BaseDirectory {
	return BaseDirectory{
		url:       url,
		destroyed: atomic.NewBool(false),
	}
}
func (dir *BaseDirectory) GetUrl() common.URL {
	return *dir.url
}
func (dir *BaseDirectory) GetDirectoryUrl() *common.URL {
	return dir.url
}

func (dir *BaseDirectory) Destroy(doDestroy func()) {
	if dir.destroyed.CAS(false, true) {
		dir.mutex.Lock()
		doDestroy()
		dir.mutex.Unlock()
	}
}

func (dir *BaseDirectory) IsAvailable() bool {
	return !dir.destroyed.Load()
}
