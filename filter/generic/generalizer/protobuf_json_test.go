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
	reqJson, err := g.Generalize(req)
	assert.Nil(t, err)
	rReq, err := g.Realize(reqJson, reflect.TypeOf(req))
	assert.Nil(t, err)
	reqObj, ok := rReq.(*RequestType)
	assert.True(t, ok)
	assert.Equal(t, req.Id, reqObj.GetId())

	resp := &ResponseType{
		Code:    200,
		Id:      1,
		Name:    "xavierniu",
		Message: "Nice to meet you",
	}
	respJson, err := g.Generalize(resp)
	assert.Nil(t, err)
	rResp, err := g.Realize(respJson, reflect.TypeOf(resp))
	assert.Nil(t, err)
	respObj, ok := rResp.(*ResponseType)
	assert.True(t, ok)
	assert.Equal(t, resp.Code, respObj.GetCode())
	assert.Equal(t, resp.Id, respObj.GetId())
	assert.Equal(t, resp.Name, respObj.GetName())
	assert.Equal(t, resp.Message, respObj.GetMessage())
}
