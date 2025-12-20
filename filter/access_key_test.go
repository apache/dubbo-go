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

package filter

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

// Mock the AccessKeyStorage interface
func TestAccessKeyPair(t *testing.T) {
	// Test case 1: Test that AccessKeyPair can be properly instantiated and fields are correctly set
	accessKeyPair := &AccessKeyPair{
		AccessKey:    "test-access-key",
		SecretKey:    "test-secret-key",
		ConsumerSide: "consumer-side",
		ProviderSide: "provider-side",
		Creator:      "creator",
		Options:      "options",
	}

	// Assert that the fields are correctly set
	assert.Equal(t, "test-access-key", accessKeyPair.AccessKey)
	assert.Equal(t, "test-secret-key", accessKeyPair.SecretKey)
	assert.Equal(t, "consumer-side", accessKeyPair.ConsumerSide)
	assert.Equal(t, "provider-side", accessKeyPair.ProviderSide)
	assert.Equal(t, "creator", accessKeyPair.Creator)
	assert.Equal(t, "options", accessKeyPair.Options)
}
