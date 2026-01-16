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
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func TestConfigCenterMetricEventType(t *testing.T) {
	event := &ConfigCenterMetricEvent{
		key:          "test-key",
		group:        "test-group",
		configCenter: Nacos,
		changeType:   remoting.EventTypeAdd,
		size:         1.0,
	}

	assert.Equal(t, constant.MetricsConfigCenter, event.Type())
}

func TestConfigCenterMetricEventGetChangeType(t *testing.T) {
	tests := []struct {
		name       string
		changeType remoting.EventType
		want       string
	}{
		{
			name:       "EventTypeAdd",
			changeType: remoting.EventTypeAdd,
			want:       "added",
		},
		{
			name:       "EventTypeDel",
			changeType: remoting.EventTypeDel,
			want:       "deleted",
		},
		{
			name:       "EventTypeUpdate",
			changeType: remoting.EventTypeUpdate,
			want:       "modified",
		},
		{
			name:       "unknown event type",
			changeType: remoting.EventType(999),
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &ConfigCenterMetricEvent{
				changeType: tt.changeType,
			}
			got := event.getChangeType()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewIncMetricEvent(t *testing.T) {
	event := NewIncMetricEvent("test-key", "test-group", remoting.EventTypeAdd, Nacos)

	assert.NotNil(t, event)
	assert.Equal(t, "test-key", event.key)
	assert.Equal(t, "test-group", event.group)
	assert.Equal(t, remoting.EventTypeAdd, event.changeType)
	assert.Equal(t, Nacos, event.configCenter)
	assert.InDelta(t, 1.0, event.size, 0.01)
}
