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

package rest

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/rest/server"
)

var restServers = make(map[string]func() server.RestServer, 8)

// SetRestServer sets the RestServer with @name
func SetRestServer(name string, fun func() server.RestServer) {
	restServers[name] = fun
}

// GetNewRestServer finds the RestServer with @name
func GetNewRestServer(name string) server.RestServer {
	if restServers[name] == nil {
		panic("restServer for " + name + " is not existing, make sure you have import the package.")
	}
	return restServers[name]()
}
