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
 * Copyright 2020 gRPC authors.
 *
 */

package client

import (
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

// WatchListener uses LDS to discover information about the provided listener.
//
// Note that during race (e.g. an xDS response is received while the user is
// calling cancel()), there's a small window where the callback can be called
// after the watcher is canceled. The caller needs to handle this case.
func (c *clientImpl) WatchListener(serviceName string, cb func(resource.ListenerUpdate, error)) (cancel func()) {
	n := resource.ParseName(serviceName)
	a, unref, err := c.findAuthority(n)
	if err != nil {
		cb(resource.ListenerUpdate{}, err)
		return func() {}
	}
	cancelF := a.watchListener(n.String(), cb)
	return func() {
		cancelF()
		unref()
	}
}

// WatchRouteConfig starts a listener watcher for the service.
//
// Note that during race (e.g. an xDS response is received while the user is
// calling cancel()), there's a small window where the callback can be called
// after the watcher is canceled. The caller needs to handle this case.
func (c *clientImpl) WatchRouteConfig(routeName string, cb func(resource.RouteConfigUpdate, error)) (cancel func()) {
	n := resource.ParseName(routeName)
	a, unref, err := c.findAuthority(n)
	if err != nil {
		cb(resource.RouteConfigUpdate{}, err)
		return func() {}
	}
	cancelF := a.watchRouteConfig(n.String(), cb)
	return func() {
		cancelF()
		unref()
	}
}

// WatchCluster uses CDS to discover information about the provided
// clusterName.
//
// WatchCluster can be called multiple times, with same or different
// clusterNames. Each call will start an independent watcher for the resource.
//
// Note that during race (e.g. an xDS response is received while the user is
// calling cancel()), there's a small window where the callback can be called
// after the watcher is canceled. The caller needs to handle this case.
func (c *clientImpl) WatchCluster(clusterName string, cb func(resource.ClusterUpdate, error)) (cancel func()) {
	n := resource.ParseName(clusterName)
	a, unref, err := c.findAuthority(n)
	if err != nil {
		cb(resource.ClusterUpdate{}, err)
		return func() {}
	}
	cancelF := a.watchCluster(n.String(), cb)
	return func() {
		cancelF()
		unref()
	}
}

// WatchEndpoints uses EDS to discover endpoints in the provided clusterName.
//
// WatchEndpoints can be called multiple times, with same or different
// clusterNames. Each call will start an independent watcher for the resource.
//
// Note that during race (e.g. an xDS response is received while the user is
// calling cancel()), there's a small window where the callback can be called
// after the watcher is canceled. The caller needs to handle this case.
func (c *clientImpl) WatchEndpoints(clusterName string, cb func(resource.EndpointsUpdate, error)) (cancel func()) {
	n := resource.ParseName(clusterName)
	a, unref, err := c.findAuthority(n)
	if err != nil {
		cb(resource.EndpointsUpdate{}, err)
		return func() {}
	}
	cancelF := a.watchEndpoints(n.String(), cb)
	return func() {
		cancelF()
		unref()
	}
}
