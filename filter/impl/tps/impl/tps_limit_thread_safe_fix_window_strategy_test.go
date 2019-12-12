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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestThreadSafeFixedWindowTpsLimitStrategyImpl_IsAllowable(t *testing.T) {
	creator := &threadSafeFixedWindowStrategyCreator{}
	strategy := creator.Create(2, 60000)
	assert.True(t, strategy.IsAllowable())
	assert.True(t, strategy.IsAllowable())
	assert.False(t, strategy.IsAllowable())

	strategy = creator.Create(2, 2000)
	assert.True(t, strategy.IsAllowable())
	assert.True(t, strategy.IsAllowable())
	assert.False(t, strategy.IsAllowable())
	time.Sleep(2100 * time.Millisecond)
	assert.True(t, strategy.IsAllowable())
	assert.True(t, strategy.IsAllowable())
	assert.False(t, strategy.IsAllowable())
}
