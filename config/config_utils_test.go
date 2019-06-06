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

package config

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestMergeValue(t *testing.T) {
	str := mergeValue("", "", "a,b")
	assert.Equal(t, "a,b", str)

	str = mergeValue("c,d", "e,f", "a,b")
	assert.Equal(t, "a,b,c,d,e,f", str)

	str = mergeValue("c,d", "e,d,f", "a,b")
	assert.Equal(t, "a,b,c,d,e,d,f", str)

	str = mergeValue("c,default,d", "-c,-a,e,f", "a,b")
	assert.Equal(t, "b,d,e,f", str)

	str = mergeValue("", "default,-b,e,f", "a,b")
	assert.Equal(t, "a,e,f", str)
}
