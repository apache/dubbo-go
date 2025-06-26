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
	"github.com/apache/dubbo-go-hessian2"

	"github.com/stretchr/testify/assert"
)

const (
	dubboParam = "org.apache.dubbo.param"
	dubboPojo  = "org.apache.dubbo.pojo"
)

type Param struct {
}

func (p *Param) JavaClassName() string {
	return dubboPojo
}

func (p *Param) JavaParamName() string {
	return dubboParam
}

type Pojo struct {
}

func (p *Pojo) JavaClassName() string {
	return dubboPojo
}

func TestGetArgType(t *testing.T) {

	t.Run("pojo", func(t *testing.T) {
		assert.Equal(t, dubboPojo, getArgType(&Pojo{}))
	})

	t.Run("param", func(t *testing.T) {
		assert.Equal(t, dubboParam, getArgType(&Param{}))
	})
}

func TestMarshalRequestWithTypedNilPointer(t *testing.T) {
	encoder := hessian.NewEncoder()
	pkg := DubboPackage{
		Service: Service{
			Path:    "test.Path",
			Version: "1.0.0",
			Method:  "Echo",
		},
		Body: &RequestPayload{
			Params: []any{(*int32)(nil)},
			Attachments: map[string]any{
				"key": "value",
			},
		},
	}

	data, err := marshalRequest(encoder, pkg)
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

func TestMarshalRequestWithNonNilPointer(t *testing.T) {
	val := int32(42)
	encoder := hessian.NewEncoder()
	pkg := DubboPackage{
		Service: Service{
			Path:    "test.Path",
			Version: "1.0.0",
			Method:  "Echo",
		},
		Body: &RequestPayload{
			Params: []any{&val},
			Attachments: map[string]any{
				"key": "value",
			},
		},
	}

	data, err := marshalRequest(encoder, pkg)
	assert.NoError(t, err)
	assert.NotNil(t, data)
}
