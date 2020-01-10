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
	"github.com/stretchr/testify/mock"
)

import (
	"github.com/apache/dubbo-go/metrics"
)

func TestDefaultCompass_Update(t *testing.T) {
	reservoir := &mockReservoir{}
	clock := &mockClock{}
	compass := NewCompass(reservoir, clock, 10, 60*time.Second, 100, 2)
	assert.Equal(t, int64(0), compass.GetCount())
	compass.Update(time.Second)
	assert.Equal(t, int64(1), compass.GetCount())
}

func TestDefaultCompass_GetSnapshot(t *testing.T) {
	snapshot := &mockSnapshot{}
	reservoir := &mockReservoir{snapshot: snapshot}
	clock := &mockClock{}
	compass := NewCompass(reservoir, clock, 10, 60*time.Second, 100, 2)
	assert.Equal(t, snapshot, compass.GetSnapshot())
}

func TestDefaultCompass_UpdateWithError(t *testing.T) {
	clock := &ManualClock{}
	compass := NewCompassWithType(metrics.BucketReservoirType, clock, 10, 60*time.Second, 10, 5)
	compass.UpdateWithError(10*time.Millisecond, true, "", "hit")
	compass.UpdateWithError(15*time.Millisecond, true, "", "")
	compass.UpdateWithError(20*time.Millisecond, false, "error1", "")
	clock.Add(60 * time.Second)

	assert.Equal(t, int64(3), compass.GetCount())
	assert.Equal(t, int64(2), compass.GetSuccessCount())
	errorCodes := compass.GetErrorCodeCounts()
	error1Counter, loaded := errorCodes["error1"]
	assert.True(t, loaded)
	assert.Equal(t, int64(1), error1Counter.GetBucketCounts()[0])

	addons := compass.GetAddonCounts()
	hitAddonCounter, loaded := addons["hit"]
	assert.True(t, loaded)
	assert.Equal(t, int64(1), hitAddonCounter.GetBucketCounts()[0])

	snapshot := compass.GetSnapshot()
	mean, _ := snapshot.GetMean()
	assert.Equal(t, float64((15 * time.Millisecond).Nanoseconds()), mean)

	compass.UpdateWithError(10*time.Millisecond, true, "", "hit")
	compass.UpdateWithError(15*time.Millisecond, true, "", "")
	compass.UpdateWithError(20*time.Millisecond, false, "error1", "")
	clock.Add(60 * time.Second)

	assert.Equal(t, int64(6), compass.GetCount())
	assert.Equal(t, int64(4), compass.GetSuccessCount())
	assert.Equal(t, int64(3), compass.GetInstantCount()[60000])

	errorCodes = compass.GetErrorCodeCounts()
	error1Counter, loaded = errorCodes["error1"]
	assert.True(t, loaded)
	assert.Equal(t, int64(1), error1Counter.GetBucketCounts()[60000])

	addons = compass.GetAddonCounts()
	hitAddonCounter, loaded = addons["hit"]
	assert.True(t, loaded)
	assert.Equal(t, int64(1), hitAddonCounter.GetBucketCounts()[0])

	assert.Equal(t, int64(2), compass.GetBucketSuccessCount().GetBucketCounts()[60000])

}

func TestDefaultCompass_Rate(t *testing.T) {
	clock := &ManualClock{}
	compass := NewCompassWithType(metrics.BucketReservoirType, clock, 10, 60*time.Second, 10, 5)
	assert.True(t, equals(0.0, compass.GetMeanRate(), 0.0001))
	assert.True(t, equals(0.0, compass.GetOneMinuteRate(), 0.0001))
	assert.True(t, equals(0.0, compass.GetFiveMinuteRate(), 0.0001))
	assert.True(t, equals(0.0, compass.GetFifteenMinuteRate(), 0.0001))
}

type mockClock struct {
	val int64
}

func (m *mockClock) GetTick() int64 {
	m.val += int64(50 * time.Millisecond)
	return m.val
}

func (m *mockClock) GetTime() int64 {
	return m.val
}

type mockSnapshot struct {
	mock.Mock
}

func (m mockSnapshot) GetValue(quantile float64) (float64, error) {
	panic("implement me")
}

func (m mockSnapshot) GetValues() ([]int64, error) {
	panic("implement me")
}

func (m mockSnapshot) Size() (int, error) {
	panic("implement me")
}

func (m mockSnapshot) GetMedian() (float64, error) {
	panic("implement me")
}

func (m mockSnapshot) Get75thPercentile() (float64, error) {
	panic("implement me")
}

func (m mockSnapshot) Get95thPercentile() (float64, error) {
	panic("implement me")
}

func (m mockSnapshot) Get98thPercentile() (float64, error) {
	panic("implement me")
}

func (m mockSnapshot) Get99thPercentile() (float64, error) {
	panic("implement me")
}

func (m mockSnapshot) Get999thPercentile() (float64, error) {
	panic("implement me")
}

func (m mockSnapshot) GetMax() (int64, error) {
	panic("implement me")
}

func (m mockSnapshot) GetMean() (float64, error) {
	panic("implement me")
}

func (m mockSnapshot) GetMin() (int64, error) {
	panic("implement me")
}

func (m mockSnapshot) GetStdDev() (float64, error) {
	panic("implement me")
}

type mockReservoir struct {
	mock.Mock
	snapshot metrics.Snapshot
	count    int
}

func (m *mockReservoir) Size() int {
	return m.count
}

func (m *mockReservoir) UpdateN(value int64) {
	m.count++
}

func (m *mockReservoir) GetSnapshot() metrics.Snapshot {
	return m.snapshot
}
