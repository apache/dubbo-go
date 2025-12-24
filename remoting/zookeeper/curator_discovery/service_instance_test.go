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

package curator_discovery

import (
	"encoding/json"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

const (
	testName    = "test-service"
	testID      = "instance-1"
	testAddress = "192.168.1.100"
	testPort    = 8080
	testTag     = "v1"
	testRegTime = int64(1609459200000)
)

func createTestInstance() *ServiceInstance {
	return &ServiceInstance{
		Name:                testName,
		ID:                  testID,
		Address:             testAddress,
		Port:                testPort,
		Payload:             map[string]string{"key": "value"},
		RegistrationTimeUTC: testRegTime,
		Tag:                 testTag,
	}
}

func TestServiceInstanceJSONSerialization(t *testing.T) {
	instance := createTestInstance()

	data, err := json.Marshal(instance)
	assert.Nil(t, err)

	var restored ServiceInstance
	err = json.Unmarshal(data, &restored)
	assert.Nil(t, err)

	assert.Equal(t, testName, restored.Name)
	assert.Equal(t, testID, restored.ID)
	assert.Equal(t, testAddress, restored.Address)
	assert.Equal(t, testPort, restored.Port)
	assert.Equal(t, testRegTime, restored.RegistrationTimeUTC)
	assert.Equal(t, testTag, restored.Tag)
}

func TestServiceInstanceJSONOmitEmpty(t *testing.T) {
	instance := &ServiceInstance{}
	data, err := json.Marshal(instance)
	assert.Nil(t, err)
	assert.Equal(t, "{}", string(data))
}
