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

package registry

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestRegistryMetricsEventType(t *testing.T) {
	event := &RegistryMetricsEvent{
		Name: Reg,
		Succ: true,
	}

	assert.Equal(t, constant.MetricsRegistry, event.Type())
}

func TestRegistryMetricsEventCostMs(t *testing.T) {
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	end := time.Now()

	event := &RegistryMetricsEvent{
		Name:  Reg,
		Start: start,
		End:   end,
	}

	cost := event.CostMs()
	assert.Greater(t, cost, 0.0)
	assert.Less(t, cost, 100.0)
}

func TestNewRegisterEvent(t *testing.T) {
	start := time.Now()
	event := NewRegisterEvent(true, start).(*RegistryMetricsEvent)

	assert.NotNil(t, event)
	assert.Equal(t, Reg, event.Name)
	assert.True(t, event.Succ)
	assert.Equal(t, start, event.Start)
	assert.True(t, event.End.After(start))
}

func TestNewSubscribeEvent(t *testing.T) {
	event := NewSubscribeEvent(true).(*RegistryMetricsEvent)

	assert.NotNil(t, event)
	assert.Equal(t, Sub, event.Name)
	assert.True(t, event.Succ)
}

func TestNewNotifyEvent(t *testing.T) {
	start := time.Now()
	event := NewNotifyEvent(start).(*RegistryMetricsEvent)

	assert.NotNil(t, event)
	assert.Equal(t, Notify, event.Name)
	assert.Equal(t, start, event.Start)
	assert.True(t, event.End.After(start))
}

func TestNewDirectoryEvent(t *testing.T) {
	event := NewDirectoryEvent("test-dir").(*RegistryMetricsEvent)

	assert.NotNil(t, event)
	assert.Equal(t, Directory, event.Name)
	assert.NotNil(t, event.Attachment)
	assert.Equal(t, "test-dir", event.Attachment["DirTyp"])
}

func TestNewServerRegisterEvent(t *testing.T) {
	start := time.Now()
	event := NewServerRegisterEvent(true, start).(*RegistryMetricsEvent)

	assert.NotNil(t, event)
	assert.Equal(t, ServerReg, event.Name)
	assert.True(t, event.Succ)
	assert.Equal(t, start, event.Start)
	assert.True(t, event.End.After(start))
}

func TestNewServerSubscribeEvent(t *testing.T) {
	event := NewServerSubscribeEvent(true).(*RegistryMetricsEvent)

	assert.NotNil(t, event)
	assert.Equal(t, ServerSub, event.Name)
	assert.True(t, event.Succ)
}
