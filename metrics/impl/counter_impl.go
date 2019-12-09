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

package impl

import (
	"sync/atomic"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

type counterImpl struct {
	count int64
}

/**
 * Implementation notes:
 * Recording the last updated time for each update is very expensive.
 * so we return 0
 */
func (c *counterImpl) LastUpdateTime() int64 {
	return 0
}

func (c *counterImpl) GetCount() int64 {
	return c.count
}

func (c *counterImpl) Inc() {
	atomic.AddInt64(&c.count, 1)
}

func (c *counterImpl) IncN(n int64) {
	atomic.AddInt64(&c.count, n)
}

func (c *counterImpl) Dec() {
	atomic.AddInt64(&c.count, -1)
}

func (c *counterImpl) DecN(n int64) {
	atomic.AddInt64(&c.count, -n)
}

func newCounter() metrics.Counter {
	return &counterImpl{}
}
