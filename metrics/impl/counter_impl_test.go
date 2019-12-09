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

func Test_counterImpl(t *testing.T) {
	instance := &counterImpl{}
	assert.Equal(t, int64(0), instance.GetCount())
	instance.Inc()
	assert.Equal(t, int64(1), instance.GetCount())
	instance.IncN(3)
	assert.Equal(t, int64(4), instance.GetCount())
	instance.Dec()
	assert.Equal(t, int64(3), instance.GetCount())
	instance.DecN(2)
	assert.Equal(t, int64(1), instance.GetCount())
}
