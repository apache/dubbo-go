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

package metadata

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metrics"
)

func TestMetadataMetricEventType(t *testing.T) {
	event := &MetadataMetricEvent{
		Name: MetadataPush,
		Succ: true,
	}

	assert.Equal(t, constant.MetricsMetadata, event.Type())
}

func TestMetadataMetricEventCostMs(t *testing.T) {
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	end := time.Now()

	event := &MetadataMetricEvent{
		Name:  MetadataPush,
		Start: start,
		End:   end,
	}

	cost := event.CostMs()
	assert.Greater(t, cost, 0.0)
	assert.Less(t, cost, 100.0) // Should be around 10ms
}

func TestNewMetadataMetricTimeEvent(t *testing.T) {
	event := NewMetadataMetricTimeEvent(MetadataPush)

	assert.NotNil(t, event)
	assert.Equal(t, MetadataPush, event.Name)
	assert.NotNil(t, event.Start)
	assert.NotNil(t, event.Attachment)
	assert.Empty(t, event.Attachment)
}

func TestMetadataMetricEventImplementsMetricsEvent(t *testing.T) {
	var _ metrics.MetricsEvent = &MetadataMetricEvent{}
}
