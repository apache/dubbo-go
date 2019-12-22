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

package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEWMA_Tick_SingleThread(t *testing.T) {
	ewma := NewOneMinuteEWMA()
	ewma.Update(3)

	ewma.Tick()

	assert.Equal(t, true, ewma.initialized)
	result := ewma.GetRate(time.Second)
	assert.Equal(t, 0.6, result )

	elapseMinute(ewma)
	result = ewma.GetRate(time.Second)
	assert.Equal(t, 0.22072766, result)

	elapseMinute(ewma)
	result = ewma.GetRate(time.Second)
	assert.Equal(t, 0.08120117, result)

}

func elapseMinute(ewma *EWMA) {
	for i := 0; i < 12; i ++ {
		ewma.Tick()
	}
}