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
 * Copyright 2021 gRPC authors.
 *
 */

// Package pubsub implements a utility type to maintain resource watchers and
// the updates.
//
// This package is designed to work with the xds resources. It could be made a
// general system that works with all types.
package pubsub

import (
	"sync"
	"time"
)

import (
	dubbogoLogger "github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
	"dubbo.apache.org/dubbo-go/v3/xds/utils/buffer"
	"dubbo.apache.org/dubbo-go/v3/xds/utils/grpcsync"
)

// Pubsub maintains resource watchers and resource updates.
//
// There can be multiple watchers for the same resource. An update to a resource
// triggers updates to all the existing watchers. Watchers can be canceled at
// any time.
type Pubsub struct {
	done               *grpcsync.Event
	logger             dubbogoLogger.Logger
	watchExpiryTimeout time.Duration

	updateCh *buffer.Unbounded // chan *watcherInfoWithUpdate
	// All the following maps are to keep the updates/metadata in a cache.
	mu          sync.Mutex
	ldsWatchers map[string]map[*watchInfo]bool
	ldsCache    map[string]resource.ListenerUpdate
	ldsMD       map[string]resource.UpdateMetadata
	rdsWatchers map[string]map[*watchInfo]bool
	rdsCache    map[string]resource.RouteConfigUpdate
	rdsMD       map[string]resource.UpdateMetadata
	cdsWatchers map[string]map[*watchInfo]bool
	cdsCache    map[string]resource.ClusterUpdate
	cdsMD       map[string]resource.UpdateMetadata
	edsWatchers map[string]map[*watchInfo]bool
	edsCache    map[string]resource.EndpointsUpdate
	edsMD       map[string]resource.UpdateMetadata
}

// New creates a new Pubsub.
func New(watchExpiryTimeout time.Duration, logger dubbogoLogger.Logger) *Pubsub {
	pb := &Pubsub{
		done:               grpcsync.NewEvent(),
		logger:             logger,
		watchExpiryTimeout: watchExpiryTimeout,

		updateCh:    buffer.NewUnbounded(),
		ldsWatchers: make(map[string]map[*watchInfo]bool),
		ldsCache:    make(map[string]resource.ListenerUpdate),
		ldsMD:       make(map[string]resource.UpdateMetadata),
		rdsWatchers: make(map[string]map[*watchInfo]bool),
		rdsCache:    make(map[string]resource.RouteConfigUpdate),
		rdsMD:       make(map[string]resource.UpdateMetadata),
		cdsWatchers: make(map[string]map[*watchInfo]bool),
		cdsCache:    make(map[string]resource.ClusterUpdate),
		cdsMD:       make(map[string]resource.UpdateMetadata),
		edsWatchers: make(map[string]map[*watchInfo]bool),
		edsCache:    make(map[string]resource.EndpointsUpdate),
		edsMD:       make(map[string]resource.UpdateMetadata),
	}
	go pb.run()
	return pb
}

// WatchListener registers a watcher for the LDS resource.
//
// It also returns whether this is the first watch for this resource.
func (pb *Pubsub) WatchListener(serviceName string, cb func(resource.ListenerUpdate, error)) (first bool, cancel func() bool) {
	wi := &watchInfo{
		c:           pb,
		rType:       resource.ListenerResource,
		target:      serviceName,
		ldsCallback: cb,
	}

	wi.expiryTimer = time.AfterFunc(pb.watchExpiryTimeout, func() {
		wi.timeout()
	})
	return pb.watch(wi)
}

// WatchRouteConfig register a watcher for the RDS resource.
//
// It also returns whether this is the first watch for this resource.
func (pb *Pubsub) WatchRouteConfig(routeName string, cb func(resource.RouteConfigUpdate, error)) (first bool, cancel func() bool) {
	wi := &watchInfo{
		c:           pb,
		rType:       resource.RouteConfigResource,
		target:      routeName,
		rdsCallback: cb,
	}

	wi.expiryTimer = time.AfterFunc(pb.watchExpiryTimeout, func() {
		wi.timeout()
	})
	return pb.watch(wi)
}

// WatchCluster register a watcher for the CDS resource.
//
// It also returns whether this is the first watch for this resource.
func (pb *Pubsub) WatchCluster(clusterName string, cb func(resource.ClusterUpdate, error)) (first bool, cancel func() bool) {
	wi := &watchInfo{
		c:           pb,
		rType:       resource.ClusterResource,
		target:      clusterName,
		cdsCallback: cb,
	}

	wi.expiryTimer = time.AfterFunc(pb.watchExpiryTimeout, func() {
		wi.timeout()
	})
	return pb.watch(wi)
}

// WatchEndpoints registers a watcher for the EDS resource.
//
// It also returns whether this is the first watch for this resource.
func (pb *Pubsub) WatchEndpoints(clusterName string, cb func(resource.EndpointsUpdate, error)) (first bool, cancel func() bool) {
	wi := &watchInfo{
		c:           pb,
		rType:       resource.EndpointsResource,
		target:      clusterName,
		edsCallback: cb,
	}

	wi.expiryTimer = time.AfterFunc(pb.watchExpiryTimeout, func() {
		wi.timeout()
	})
	return pb.watch(wi)
}

// Close closes the pubsub.
func (pb *Pubsub) Close() {
	if pb.done.HasFired() {
		return
	}
	pb.done.Fire()
}

// run is a goroutine for all the callbacks.
//
// Callback can be called in watch(), if an item is found in cache. Without this
// goroutine, the callback will be called inline, which might cause a deadlock
// in user's code. Callbacks also cannot be simple `go callback()` because the
// order matters.
func (pb *Pubsub) run() {
	for {
		select {
		case t := <-pb.updateCh.Get():
			pb.updateCh.Load()
			if pb.done.HasFired() {
				return
			}
			pb.callCallback(t.(*watcherInfoWithUpdate))
		case <-pb.done.Done():
			return
		}
	}
}
