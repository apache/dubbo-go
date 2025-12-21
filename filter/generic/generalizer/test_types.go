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

// RequestType is a test type for generalizer tests
type RequestType struct {
	Id int64 `json:"id"`
}

func (r *RequestType) GetId() int64 {
	return r.Id
}

// ResponseType is a test type for generalizer tests
type ResponseType struct {
	Code    int64  `json:"code"`
	Id      int64  `json:"id"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

func (r *ResponseType) GetCode() int64 {
	return r.Code
}

func (r *ResponseType) GetId() int64 {
	return r.Id
}

func (r *ResponseType) GetName() string {
	return r.Name
}

func (r *ResponseType) GetMessage() string {
	return r.Message
}
