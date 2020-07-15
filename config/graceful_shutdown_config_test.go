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

package config

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestShutdownConfigGetTimeout(t *testing.T) {
	config := ShutdownConfig{}
	assert.False(t, config.RejectRequest)
	assert.False(t, config.RequestsFinished)

	config = ShutdownConfig{
		Timeout:     "12x",
		StepTimeout: "34a",
	}

	assert.Equal(t, 60*time.Second, config.GetTimeout())
	assert.Equal(t, 10*time.Second, config.GetStepTimeout())

	config = ShutdownConfig{
		Timeout:     "34ms",
		StepTimeout: "79ms",
	}

	assert.Equal(t, 34*time.Millisecond, config.GetTimeout())
	assert.Equal(t, 79*time.Millisecond, config.GetStepTimeout())
}
