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

import (
	"dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc"
	adasvcfilter "dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter"
)

func TestDoesAdaptiveServiceReachLimitationWithNil(t *testing.T) {
	result := DoesAdaptiveServiceReachLimitation(nil)
	assert.False(t, result)
}

func TestDoesAdaptiveServiceReachLimitationWithMatchingError(t *testing.T) {
	err := errors.New(ReachLimitationErrorString)
	result := DoesAdaptiveServiceReachLimitation(err)
	assert.True(t, result)
}

func TestDoesAdaptiveServiceReachLimitationWithNonMatchingError(t *testing.T) {
	err := errors.New("some other error")
	result := DoesAdaptiveServiceReachLimitation(err)
	assert.False(t, result)
}

func TestDoesAdaptiveServiceReachLimitationWithPartialMatch(t *testing.T) {
	err := errors.New(adaptivesvc.ErrAdaptiveSvcInterrupted.Error())
	result := DoesAdaptiveServiceReachLimitation(err)
	assert.False(t, result)
}

func TestIsAdaptiveServiceFailedWithNil(t *testing.T) {
	result := IsAdaptiveServiceFailed(nil)
	assert.False(t, result)
}

func TestIsAdaptiveServiceFailedWithMatchingError(t *testing.T) {
	err := errors.New(adaptivesvc.ErrAdaptiveSvcInterrupted.Error() + ": some details")
	result := IsAdaptiveServiceFailed(err)
	assert.True(t, result)
}

func TestIsAdaptiveServiceFailedWithExactError(t *testing.T) {
	err := errors.New(adaptivesvc.ErrAdaptiveSvcInterrupted.Error())
	result := IsAdaptiveServiceFailed(err)
	assert.True(t, result)
}

func TestIsAdaptiveServiceFailedWithNonMatchingError(t *testing.T) {
	err := errors.New("completely different error")
	result := IsAdaptiveServiceFailed(err)
	assert.False(t, result)
}

func TestIsAdaptiveServiceFailedWithReachLimitationError(t *testing.T) {
	err := errors.New(ReachLimitationErrorString)
	result := IsAdaptiveServiceFailed(err)
	assert.True(t, result)
}

func TestReachLimitationErrorStringFormat(t *testing.T) {
	expected := adaptivesvc.ErrAdaptiveSvcInterrupted.Error() + ": " + adasvcfilter.ErrReachLimitation.Error()
	assert.Equal(t, expected, ReachLimitationErrorString)
}
