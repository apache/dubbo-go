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

package client

import (
	"net/http"
	"time"
)

// RestOptions
type RestOptions struct {
	RequestTimeout time.Duration
	ConnectTimeout time.Duration
}

// RestClientRequest
type RestClientRequest struct {
	Header      http.Header
	Location    string
	Path        string
	Method      string
	PathParams  map[string]string
	QueryParams map[string]string
	Body        interface{}
}

// RestClient user can implement this client interface to send request
type RestClient interface {
	Do(request *RestClientRequest, res interface{}) error
}
