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

type RequestBody struct {
	Required bool                  `json:"required,omitempty"`
	Content  map[string]*MediaType `json:"content"`
}

func NewRequestBody() *RequestBody {
	return &RequestBody{
		Content: make(map[string]*MediaType),
	}
}

func (r *RequestBody) SetRequired(required bool) *RequestBody {
	r.Required = required
	return r
}

func (r *RequestBody) AddContent(mediaType string, content *MediaType) *RequestBody {
	if r.Content == nil {
		r.Content = make(map[string]*MediaType)
	}
	r.Content[mediaType] = content
	return r
}

func (r *RequestBody) GetOrAddContent(mediaType string) *MediaType {
	if r.Content == nil {
		r.Content = make(map[string]*MediaType)
	}
	if _, ok := r.Content[mediaType]; !ok {
		r.Content[mediaType] = NewMediaType()
	}
	return r.Content[mediaType]
}
