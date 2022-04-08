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

/*
 *
 * Copyright 2019 gRPC authors.
 *
 */

package client

import (
	"dubbo.apache.org/dubbo-go/v3/xds/client/load"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

// ReportLoad starts an load reporting stream to the given server. If the server
// is not an empty string, and is different from the management server, a new
// ClientConn will be created.
//
// The same options used for creating the Client will be used (including
// NodeProto, and dial options if necessary).
//
// It returns a Store for the user to report loads, a function to cancel the
// load reporting stream.
func (c *clientImpl) ReportLoad(server string) (*load.Store, func()) {
	// TODO: load reporting with federation also needs find the authority for
	// this server first, then reports load to it. Currently always report to
	// the default authority. This is needed to avoid a nil pointer panic.
	a, unref, err := c.findAuthority(resource.ParseName(""))
	if err != nil {
		return nil, func() {}
	}
	store, cancelF := a.reportLoad(server)
	return store, func() {
		cancelF()
		unref()
	}
}
