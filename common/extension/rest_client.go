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

package extension

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/rest/client"
)

var restClients = make(map[string]func(restOptions *client.RestOptions) client.RestClient, 8)

// SetRestClient sets the RestClient with @name
func SetRestClient(name string, fun func(_ *client.RestOptions) client.RestClient) {
	restClients[name] = fun
}

// GetNewRestClient finds the RestClient with @name
func GetNewRestClient(name string, restOptions *client.RestOptions) client.RestClient {
	if restClients[name] == nil {
		panic("restClient for " + name + " is not existing, make sure you have import the package.")
	}
	return restClients[name](restOptions)
}
