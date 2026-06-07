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

package report

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestMergeServiceAppMapping(t *testing.T) {
	tests := []struct {
		name        string
		oldVal      string
		application string
		wantVal     string
		wantChanged bool
	}{
		{"empty old value", "", "appA", "appA", true},
		{"append new app", "appA", "appB", "appA,appB", true},
		{"already present", "appA,appB", "appA", "appA,appB", false},
		{"already present last", "appA,appB", "appB", "appA,appB", false},
		// regression: strings.Contains would treat "order" as already present in
		// "order-service" and never write it. Whole-element matching must not.
		{"substring is not membership", "order-service", "order", "order-service,order", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := MergeServiceAppMapping(tt.oldVal, tt.application)
			assert.Equal(t, tt.wantVal, got)
			assert.Equal(t, tt.wantChanged, changed)
		})
	}
}

func TestDecodeServiceAppNames(t *testing.T) {
	// blank value yields an empty set, not a set containing one empty string.
	assert.True(t, DecodeServiceAppNames("").Empty())

	set := DecodeServiceAppNames("appA,appB")
	assert.Equal(t, 2, set.Size())
	assert.True(t, set.Contains("appA"))
	assert.True(t, set.Contains("appB"))

	// empty elements from stray separators are skipped.
	set = DecodeServiceAppNames("appA,,appB,")
	assert.Equal(t, 2, set.Size())
	assert.False(t, set.Contains(""))
}
