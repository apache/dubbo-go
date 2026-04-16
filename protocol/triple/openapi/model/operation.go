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

package model

type Operation struct {
	Tags        []string                `json:"tags,omitempty"`
	OperationId string                  `json:"operationId,omitempty"`
	RequestBody *RequestBody            `json:"requestBody,omitempty"`
	Responses   map[string]*ApiResponse `json:"responses,omitempty"`

	HttpMethod string `json:"-"`
	GoMethod   string `json:"-"`
}

func NewOperation() *Operation {
	return &Operation{
		Responses: make(map[string]*ApiResponse),
	}
}

func (o *Operation) SetOperationId(id string) *Operation {
	o.OperationId = id
	return o
}

func (o *Operation) SetHttpMethod(method string) *Operation {
	o.HttpMethod = method
	return o
}

func (o *Operation) SetGoMethod(method string) *Operation {
	o.GoMethod = method
	return o
}

func (o *Operation) AddTag(tag string) *Operation {
	o.Tags = append(o.Tags, tag)
	return o
}

func (o *Operation) SetRequestBody(body *RequestBody) *Operation {
	o.RequestBody = body
	return o
}

func (o *Operation) GetOrAddResponse(code string) *ApiResponse {
	if o.Responses == nil {
		o.Responses = make(map[string]*ApiResponse)
	}
	if _, ok := o.Responses[code]; !ok {
		o.Responses[code] = NewApiResponse()
	}
	return o.Responses[code]
}
