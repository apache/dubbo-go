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

package client

import (
	"errors"
	"fmt"
)

import (
	dubbogoLogger "github.com/dubbogo/gost/log/logger"

	_struct "github.com/golang/protobuf/ptypes/struct"
)

import (
	"dubbo.apache.org/dubbo-go/v3/xds/client/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/xds/client/load"
	"dubbo.apache.org/dubbo-go/v3/xds/client/pubsub"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

const federationScheme = "xdstp"

// findAuthority returns the authority for this name. If it doesn't already
// exist, one will be created.
//
// Note that this doesn't always create new authority. authorities with the same
// config but different names are shared.
//
// The returned unref function must be called when the caller is done using this
// authority, without holding c.authorityMu.
//
// Caller must not hold c.authorityMu.
func (c *clientImpl) findAuthority(n *resource.Name) (_ *authority, unref func(), _ error) {
	scheme, authority := n.Scheme, n.Authority

	c.authorityMu.Lock()
	defer c.authorityMu.Unlock()
	if c.done.HasFired() {
		return nil, nil, errors.New("the xds-client is closed")
	}

	config := c.config.XDSServer
	if scheme == federationScheme {
		cfg, ok := c.config.Authorities[authority]
		if !ok {
			return nil, nil, fmt.Errorf("xds: failed to find authority %q", authority)
		}
		config = cfg.XDSServer
	}

	a, err := c.newAuthority(config)
	if err != nil {
		dubbogoLogger.Errorf(`[XDS Authority] new authority failed with error = %s, please makesure you have imported 
	_ "dubbo.apache.org/dubbo-go/v3/xds/client/controller/version/v2"
	_ "dubbo.apache.org/dubbo-go/v3/xds/client/controller/version/v3"`, err)
		return nil, nil, fmt.Errorf("xds: failed to connect to the control plane for authority %q: %v", authority, err)
	}
	// All returned authority from this function will be used by a watch,
	// holding the ref here.
	//
	// Note that this must be done while c.authorityMu is held, to avoid the
	// race that an authority is returned, but before the watch starts, the
	// old last watch is canceled (in another goroutine), causing this
	// authority to be removed, and then a watch will start on a removed
	// authority.
	//
	// unref() will be done when the watch is canceled.
	a.ref()
	return a, func() { c.unrefAuthority(a) }, nil
}

// newAuthority creates a new authority for the config. But before that, it
// checks the cache to see if an authority for this config already exists.
//
// caller must hold c.authorityMu
func (c *clientImpl) newAuthority(config *bootstrap.ServerConfig) (_ *authority, retErr error) {
	// First check if there's already an authority for this config. If found, it
	// means this authority is used by other watches (could be the same
	// authority name, or a different authority name but the same server
	// config). Return it.
	configStr := config.String()
	if a, ok := c.authorities[configStr]; ok {
		return a, nil
	}
	// Second check if there's an authority in the idle cache. If found, it
	// means this authority was created, but moved to the idle cache because the
	// watch was canceled. Move it from idle cache to the authority cache, and
	// return.
	if old, ok := c.idleAuthorities.Remove(configStr); ok {
		oldA, _ := old.(*authority)
		if oldA != nil {
			c.authorities[configStr] = oldA
			return oldA, nil
		}
	}

	// Make a new authority since there's no existing authority for this config.
	ret := &authority{config: config, pubsub: pubsub.New(c.watchExpiryTimeout, c.logger)}
	defer func() {
		if retErr != nil {
			ret.close()
		}
	}()
	ctr, err := newController(config, ret.pubsub, c.updateValidator, c.logger)
	if err != nil {
		return nil, err
	}
	ret.controller = ctr
	// Add it to the cache, so it will be reused.
	c.authorities[configStr] = ret
	return ret, nil
}

// unrefAuthority unrefs the authority. It also moves the authority to idle
// cache if it's ref count is 0.
//
// This function doesn't need to called explicitly. It's called by the returned
// unref from findAuthority().
//
// Caller must not hold c.authorityMu.
func (c *clientImpl) unrefAuthority(a *authority) {
	c.authorityMu.Lock()
	defer c.authorityMu.Unlock()
	if a.unref() > 0 {
		return
	}
	configStr := a.config.String()
	delete(c.authorities, configStr)
	c.idleAuthorities.Add(configStr, a, func() {
		a.close()
	})
}

// authority is a combination of pubsub and the controller for this authority.
//
// Note that it might make sense to use one pubsub for all the resources (for
// all the controllers). One downside is the handling of StoW APIs (LDS/CDS).
// These responses contain all the resources from that control plane, so pubsub
// will need to keep lists of resources from each control plane, to know what
// are removed.
type authority struct {
	config     *bootstrap.ServerConfig
	pubsub     *pubsub.Pubsub
	controller controllerInterface
	refCount   int
}

func (a *authority) SetMetadata(m *_struct.Struct) error {
	return a.controller.SetMetadata(m)
}

// caller must hold parent's authorityMu.
func (a *authority) ref() {
	a.refCount++
}

// caller must hold parent's authorityMu.
func (a *authority) unref() int {
	a.refCount--
	return a.refCount
}

func (a *authority) close() {
	if a.pubsub != nil {
		a.pubsub.Close()
	}
	if a.controller != nil {
		a.controller.Close()
	}
}

func (a *authority) watchListener(serviceName string, cb func(resource.ListenerUpdate, error)) (cancel func()) {
	first, cancelF := a.pubsub.WatchListener(serviceName, cb)
	if first {
		a.controller.AddWatch(resource.ListenerResource, serviceName)
	}
	return func() {
		if cancelF() {
			a.controller.RemoveWatch(resource.ListenerResource, serviceName)
		}
	}
}

func (a *authority) watchRouteConfig(routeName string, cb func(resource.RouteConfigUpdate, error)) (cancel func()) {
	first, cancelF := a.pubsub.WatchRouteConfig(routeName, cb)
	if first {
		a.controller.AddWatch(resource.RouteConfigResource, routeName)
	}
	return func() {
		if cancelF() {
			a.controller.RemoveWatch(resource.RouteConfigResource, routeName)
		}
	}
}

func (a *authority) watchCluster(clusterName string, cb func(resource.ClusterUpdate, error)) (cancel func()) {
	first, cancelF := a.pubsub.WatchCluster(clusterName, cb)
	if first {
		a.controller.AddWatch(resource.ClusterResource, clusterName)
	}
	return func() {
		if cancelF() {
			a.controller.RemoveWatch(resource.ClusterResource, clusterName)
		}
	}
}

func (a *authority) watchEndpoints(clusterName string, cb func(resource.EndpointsUpdate, error)) (cancel func()) {
	first, cancelF := a.pubsub.WatchEndpoints(clusterName, cb)
	if first {
		a.controller.AddWatch(resource.EndpointsResource, clusterName)
	}
	return func() {
		if cancelF() {
			a.controller.RemoveWatch(resource.EndpointsResource, clusterName)
		}
	}
}

func (a *authority) reportLoad(server string) (*load.Store, func()) {
	return a.controller.ReportLoad(server)
}

func (a *authority) dump(t resource.ResourceType) map[string]resource.UpdateWithMD {
	return a.pubsub.Dump(t)
}
