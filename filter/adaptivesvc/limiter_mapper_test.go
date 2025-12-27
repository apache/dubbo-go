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

package adaptivesvc

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter"
)

func TestLimiterMapper_newAndSetMethodLimiter(t *testing.T) {
	// Initialize the limiterMapper and mock URL
	mapper := newLimiterMapper()
	url := &common.URL{Path: "/testService"}
	methodName := "testMethod"

	// Test creating a new limiter
	l, err := mapper.newAndSetMethodLimiter(url, methodName, limiter.HillClimbingLimiter)
	require.NoError(t, err)
	assert.NotNil(t, l)

	// Test that the same limiter is returned if already created
	l2, err := mapper.newAndSetMethodLimiter(url, methodName, limiter.HillClimbingLimiter)
	require.NoError(t, err)
	assert.Same(t, l, l2)
}

func TestLimiterMapper_getMethodLimiter(t *testing.T) {
	// Initialize the limiterMapper and mock URL
	mapper := newLimiterMapper()
	url := &common.URL{Path: "/testService"}
	methodName := "testMethod"

	// Add a limiter to the mapper
	_, err := mapper.newAndSetMethodLimiter(url, methodName, limiter.HillClimbingLimiter)
	require.NoError(t, err)

	// Test getting an existing limiter
	l, err := mapper.getMethodLimiter(url, methodName)
	require.NoError(t, err)
	assert.NotNil(t, l)

	// Test getting a limiter that does not exist
	url2 := &common.URL{Path: "/testService2"}
	l, err = mapper.getMethodLimiter(url2, "nonExistentMethod")
	require.Error(t, err)
	assert.Nil(t, l)
	assert.Equal(t, ErrLimiterNotFoundOnMapper, err)
}
