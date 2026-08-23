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

package dubboutil

import (
	"encoding/json"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONHelpersUseNumber(t *testing.T) {
	encoded, err := EncodeJSON(map[string]any{
		"int":     42,
		"float":   1.25,
		"nullVal": nil,
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"int":42,"float":1.25,"nullVal":null}`, string(encoded))

	var decoded map[string]any
	err = DecodeJSON(encoded, &decoded)
	require.NoError(t, err)
	assert.Equal(t, json.Number("42"), decoded["int"])
	assert.Equal(t, json.Number("1.25"), decoded["float"])
	assert.Nil(t, decoded["nullVal"])
}
