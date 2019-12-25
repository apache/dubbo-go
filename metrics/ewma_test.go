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
	"math"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestEWMA_Tick_OneMinute(t *testing.T) {
	ewma := NewOneMinuteEWMA()
	ewma.Update(3)

	ewma.Tick()

	assert.Equal(t, true, ewma.initialized)
	result := ewma.GetRate(time.Second)
	assert.True(t, equals(0.6, result, 0.000001))

	// copy from Dubbo and then calculate manually... Those data are correct.
	// Or you can calculate manually by yourself.
	expectedArray := []float64{
		0.22072766, 0.08120117, 0.02987224, 0.01098938, 0.00404277,
		0.00148725, 0.00054713, 0.00020128, 0.00007405, 0.00002724,
		0.00001002, 0.00000369, 0.00000136, 0.00000050, 0.00000018,
	}

	for _, expected := range expectedArray {
		elapseMinute(ewma)
		result = ewma.GetRate(time.Second)
		assert.True(t, equals(expected, result, 0.000001))
	}
}

func TestEWMA_Tick_FiveMinute(t *testing.T) {
	ewma := NewFiveMinutesEWMA()
	ewma.Update(3)

	ewma.Tick()

	assert.Equal(t, true, ewma.initialized)
	result := ewma.GetRate(time.Second)
	assert.True(t, equals(0.6, result, 0.000001))

	// copy from Dubbo and then calculate manually... Those data are correct.
	// Or you can calculate manually by yourself.
	expectedArray := []float64{
		0.49123845, 0.40219203, 0.32928698, 0.26959738, 0.22072766,
		0.18071653, 0.14795818, 0.12113791, 0.09917933, 0.08120117,
		0.06648190, 0.05443077, 0.04456415, 0.03648604, 0.02987224,
	}

	for _, expected := range expectedArray {
		elapseMinute(ewma)
		result = ewma.GetRate(time.Second)
		assert.True(t, equals(expected, result, 0.000001))
	}
}

func TestEWMA_Tick_FifteenMinute(t *testing.T) {
	ewma := NewFifteenMinutesEWMA()
	ewma.Update(3)

	ewma.Tick()

	assert.Equal(t, true, ewma.initialized)
	result := ewma.GetRate(time.Second)
	assert.True(t, equals(0.6, result, 0.000001))

	// copy from Dubbo and then calculate manually... Those data are correct.
	// Or you can calculate manually by yourself.
	expectedArray := []float64{
		0.56130419, 0.52510399, 0.49123845, 0.45955700, 0.42991879,
		0.40219203, 0.37625345, 0.35198773, 0.32928698, 0.30805027,
		0.28818318, 0.26959738, 0.25221023, 0.23594443, 0.22072766,
	}

	for _, expected := range expectedArray {
		elapseMinute(ewma)
		result = ewma.GetRate(time.Second)
		assert.True(t, equals(expected, result, 0.000001))
	}
}

// compare two float numbers
func equals(expected float64, actual float64, delta float64) bool {
	return math.Abs(actual-expected) < delta
}

func elapseMinute(ewma *EWMA) {
	for i := 0; i < 12; i++ {
		ewma.Tick()
	}
}
