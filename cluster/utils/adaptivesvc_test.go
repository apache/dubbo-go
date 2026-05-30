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

package utils

import (
	"errors"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestDoesAdaptiveServiceReachLimitation(t *testing.T) {
	assert.False(t, DoesAdaptiveServiceReachLimitation(nil))
	assert.True(t, DoesAdaptiveServiceReachLimitation(errors.New(ReachLimitationErrorString)))
	assert.True(t, DoesAdaptiveServiceReachLimitation(errors.New("unknown: "+ReachLimitationErrorString)))
	assert.False(t, DoesAdaptiveServiceReachLimitation(errors.New("adaptive service interrupted: another error")))
}

func TestIsAdaptiveServiceFailed(t *testing.T) {
	assert.False(t, IsAdaptiveServiceFailed(nil))
	assert.True(t, IsAdaptiveServiceFailed(errors.New("adaptive service interrupted: reach limitation")))
	assert.True(t, IsAdaptiveServiceFailed(errors.New("unknown: adaptive service interrupted: reach limitation")))
	assert.False(t, IsAdaptiveServiceFailed(errors.New("unavailable: network error")))
}
