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
)
import (
	"github.com/stretchr/testify/assert"
)

func TestNewUniformReservoir100OutOf1000(t *testing.T) {
	reservoir := NewUniformReservoir(100)
	for i := 0; i < 1000; i++ {
		reservoir.UpdateN(int64(i))
	}

	snapshot := reservoir.GetSnapshot()
	assert.Equal(t, 100, reservoir.Size())
	size, _ := snapshot.Size()
	assert.Equal(t, 100, size)
	values, _ := snapshot.GetValues()
	for _, value := range values {
		assert.True(t, value < 1000)
		assert.True(t, value >= 0)
	}
}
