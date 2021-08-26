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

package judger

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
)

func TestListDoubleMatchJudger_Judge(t *testing.T) {
	assert.True(t, newListDoubleMatchJudger(&config.ListDoubleMatch{
		Oneof: []*config.DoubleMatch{
			{
				Exact: 3.14,
			},
			{
				Range: &config.DoubleRangeMatch{
					Start: 1.5,
					End:   1.9,
				},
			},
			{
				Exact: 1.3,
			},
		},
	}).Judge(1.3))

	assert.False(t, newListDoubleMatchJudger(&config.ListDoubleMatch{
		Oneof: []*config.DoubleMatch{
			{
				Exact: 3.14,
			},
			{
				Range: &config.DoubleRangeMatch{
					Start: 1.5,
					End:   1.9,
				},
			},
			{
				Exact: 1.2,
			},
		},
	}).Judge(1.3))

	assert.True(t, newListDoubleMatchJudger(&config.ListDoubleMatch{
		Oneof: []*config.DoubleMatch{
			{
				Exact: 3.14,
			},
			{
				Range: &config.DoubleRangeMatch{
					Start: 1.2,
					End:   1.9,
				},
			},
			{
				Exact: 1.4,
			},
		},
	}).Judge(1.3))

	assert.False(t, newListDoubleMatchJudger(&config.ListDoubleMatch{
		Oneof: []*config.DoubleMatch{
			{
				Exact: 3.14,
			},
			{
				Range: &config.DoubleRangeMatch{
					Start: 1.5,
					End:   1.9,
				},
			},
			{
				Exact: 1.0,
			},
		},
	}).Judge(1.3))
}
