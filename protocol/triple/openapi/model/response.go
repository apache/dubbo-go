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

type ApiResponse struct {
	Description string                `json:"description"`
	Content     map[string]*MediaType `json:"content,omitempty"`
}

func NewApiResponse() *ApiResponse {
	return &ApiResponse{
		Content: make(map[string]*MediaType),
	}
}

func (a *ApiResponse) SetDescription(desc string) *ApiResponse {
	a.Description = desc
	return a
}

func (a *ApiResponse) AddContent(mediaType string, content *MediaType) *ApiResponse {
	if a.Content == nil {
		a.Content = make(map[string]*MediaType)
	}
	a.Content[mediaType] = content
	return a
}

func (a *ApiResponse) GetOrAddContent(mediaType string) *MediaType {
	if a.Content == nil {
		a.Content = make(map[string]*MediaType)
	}
	if _, ok := a.Content[mediaType]; !ok {
		a.Content[mediaType] = NewMediaType()
	}
	return a.Content[mediaType]
}
