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

package parser

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigurationParser_Parser(t *testing.T) {
	parser := &DefaultConfigurationParser{}
	m, err := parser.Parse("dubbo.registry.address=172.0.0.1\ndubbo.registry.name=test")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(m))
	assert.Equal(t, "172.0.0.1", m["dubbo.registry.address"])
}
