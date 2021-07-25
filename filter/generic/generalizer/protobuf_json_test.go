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

package generalizer

import (
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestProtobufJsonGeneralizer(t *testing.T) {
	g := GetProtobufJsonGeneralizer()

	req := &RequestType{
		Id: 1,
	}
	reqjson, err := g.Generalize(req)
	assert.Nil(t, err)
	rreq, err := g.Realize(reqjson, reflect.TypeOf(req))
	assert.Nil(t, err)
	reqobj, ok := rreq.(*RequestType)
	assert.True(t, ok)
	assert.Equal(t, req.Id, reqobj.GetId())

	resp := &ResponseType{
		Code:    200,
		Id:      1,
		Name:    "xavierniu",
		Message: "Nice to meet you",
	}
	respjson, err := g.Generalize(resp)
	assert.Nil(t, err)
	rresp, err := g.Realize(respjson, reflect.TypeOf(resp))
	assert.Nil(t, err)
	respobj, ok := rresp.(*ResponseType)
	assert.True(t, ok)
	assert.Equal(t, resp.Code, respobj.GetCode())
	assert.Equal(t, resp.Id, respobj.GetId())
	assert.Equal(t, resp.Name, respobj.GetName())
	assert.Equal(t, resp.Message, respobj.GetMessage())
}
