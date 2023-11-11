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

package client

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestWithURL(t *testing.T) {
	tests := []struct {
		opts    []ClientOption
		justify func(t *testing.T, opts *ClientOptions)
	}{
		{
			opts: []ClientOption{
				WithURL("127.0.0.1:20000"),
			},
			justify: func(t *testing.T, opts *ClientOptions) {
				urls := opts.urls
				assert.Equal(t, 1, len(urls))
				assert.Equal(t, "tri", urls[0].Protocol)
			},
		},
		{
			opts: []ClientOption{
				WithURL("tri://127.0.0.1:20000"),
			},
			justify: func(t *testing.T, opts *ClientOptions) {
				urls := opts.urls
				assert.Equal(t, 1, len(urls))
				assert.Equal(t, "tri", urls[0].Protocol)
			},
		},
	}

	for _, test := range tests {
		newOpts := defaultClientOptions()
		assert.Nil(t, newOpts.init(test.opts...))
		assert.Nil(t, newOpts.processURL(&common.URL{}))
		test.justify(t, newOpts)
	}
}
