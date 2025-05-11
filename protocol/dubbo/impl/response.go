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

type ResponsePayload struct {
	RspObj      any
	Exception   error
	Attachments map[string]any
}

// NewResponsePayload create a new ResponsePayload
func NewResponsePayload(rspObj any, exception error, attachments map[string]any) *ResponsePayload {
	if attachments == nil {
		attachments = make(map[string]any)
	}
	return &ResponsePayload{
		RspObj:      rspObj,
		Exception:   exception,
		Attachments: attachments,
	}
}

func EnsureResponsePayload(body any) *ResponsePayload {
	if res, ok := body.(*ResponsePayload); ok {
		return res
	}
	if exp, ok := body.(error); ok {
		return NewResponsePayload(nil, exp, nil)
	}
	return NewResponsePayload(body, nil, nil)
}
