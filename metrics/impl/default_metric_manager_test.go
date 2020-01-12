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
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

func TestDefaultMetricManager_GetFastCompass(t *testing.T) {
	manager := newDefaultMetricManager()
	assert.True(t, manager.IsEnable())
	manager.SetEnable(false)
	assert.False(t, manager.IsEnable())

	groupName := "TestGroup"
	name := metrics.NewMetricName("Test", nil, metrics.Minor)
	fastCompass := manager.GetFastCompass(groupName, name)
	assert.Equal(t, nopFastCompass, fastCompass)

	manager.SetEnable(true)
	fastCompass = manager.GetFastCompass(groupName, name)
	fastCompass1 := manager.GetFastCompass(groupName, name)
	assert.Equal(t, fastCompass, fastCompass1)

}
