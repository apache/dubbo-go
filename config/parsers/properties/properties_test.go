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

package properties

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestProperties_Marshal(t *testing.T) {
	p := &Properties{}
	bytes, err := p.Marshal(map[string]interface{}{})
	assert.NoError(t, err)
	assert.Nil(t, bytes)
}

func TestProperties_Unmarshal(t *testing.T) {
	str := `Name=dubbo-go
module=local`
	p := &Properties{}
	unmarshal, err := p.Unmarshal([]byte(str))
	assert.NoError(t, err)
	assert.Equal(t, unmarshal["Name"], "dubbo-go")
}
