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

package match

import (
	"context"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
)

func TestIsMatchInternalPattern(t *testing.T) {
	assert.Equal(t, true, IsMatchInternalPattern("*", "value"))
	assert.Equal(t, true, IsMatchInternalPattern("", ""))
	assert.Equal(t, false, IsMatchInternalPattern("", "value"))
	assert.Equal(t, true, IsMatchInternalPattern("value", "value"))
	assert.Equal(t, true, IsMatchInternalPattern("v*", "value"))
	assert.Equal(t, true, IsMatchInternalPattern("*ue", "value"))
	assert.Equal(t, true, IsMatchInternalPattern("*e", "value"))
	assert.Equal(t, true, IsMatchInternalPattern("v*e", "value"))
}

func TestIsMatchGlobPattern(t *testing.T) {
	url, _ := common.NewURL(context.TODO(), "dubbo://localhost:8080/Foo?key=v*e")
	assert.Equal(t, true, IsMatchGlobalPattern("$key", "value", &url))
}
