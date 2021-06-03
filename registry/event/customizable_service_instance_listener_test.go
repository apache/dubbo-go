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

package event

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func TestGetCustomizableServiceInstanceListener(t *testing.T) {

	cus := GetCustomizableServiceInstanceListener()

	assert.Equal(t, 9999, cus.GetPriority())

	extension.AddCustomizers(&mockCustomizer{})

	err := cus.OnEvent(&mockEvent{})
	assert.Nil(t, err)
	err = cus.OnEvent(NewServiceInstancePreRegisteredEvent("hello", createInstance()))
	assert.Nil(t, err)

	tp := cus.GetEventType()
	assert.NotNil(t, tp)
}

type mockEvent struct{}

func (m *mockEvent) String() string {
	panic("implement me")
}

func (m *mockEvent) GetSource() interface{} {
	panic("implement me")
}

func (m *mockEvent) GetTimestamp() time.Time {
	panic("implement me")
}

type mockCustomizer struct{}

func (m *mockCustomizer) GetPriority() int {
	return 0
}

func (m *mockCustomizer) Customize(instance registry.ServiceInstance) {
}
