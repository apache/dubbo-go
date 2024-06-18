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

package triple

import (
	"net/http"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func Test_generateAttachments(t *testing.T) {
	tests := []struct {
		desc   string
		input  func() http.Header
		expect func(t *testing.T, res map[string]interface{})
	}{
		{
			desc: "empty header",
			input: func() http.Header {
				return http.Header{}
			},
			expect: func(t *testing.T, res map[string]interface{}) {
				assert.Zero(t, len(res))
			},
		},
		{
			desc: "normal header with lowercase keys",
			input: func() http.Header {
				header := make(http.Header)
				header.Set("key1", "val1")
				header.Set("key2", "val2_1")
				header.Add("key2", "val2_2")
				return header
			},
			expect: func(t *testing.T, res map[string]interface{}) {
				assert.Equal(t, 2, len(res))
				assert.Equal(t, []string{"val1"}, res["key1"])
				assert.Equal(t, []string{"val2_1", "val2_2"}, res["key2"])
			},
		},
		{
			desc: "normal header with uppercase keys",
			input: func() http.Header {
				header := make(http.Header)
				header.Set("Key1", "val1")
				header.Set("Key2", "val2_1")
				header.Add("Key2", "val2_2")
				return header
			},
			expect: func(t *testing.T, res map[string]interface{}) {
				assert.Equal(t, 2, len(res))
				assert.Equal(t, []string{"val1"}, res["key1"])
				assert.Equal(t, []string{"val2_1", "val2_2"}, res["key2"])
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			atta := generateAttachments(test.input())
			test.expect(t, atta)
		})
	}
}
