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

package kubernetes

import (
	"context"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

import (
	"github.com/apache/dubbo-go/common/logger"
)

type Client struct {
	lock sync.RWMutex

	// manage the  client lifecycle
	ctx    context.Context
	cancel context.CancelFunc

	controller *dubboRegistryController
}

// newClient
// new a client for registry
func newClient(namespace string) (*Client, error) {

	ctx, cancel := context.WithCancel(context.Background())
	controller, err := newDubboRegistryController(ctx)
	if err != nil {
		return nil, perrors.WithMessage(err, "new dubbo-registry controller")
	}

	c := &Client{
		ctx:        ctx,
		cancel:     cancel,
		controller: controller,
	}

	c.controller.Run()

	return c, nil
}

// Create
// create k/v pair in watcher-set
func (c *Client) Create(k, v string) error {

	// the read current pod must be lock, protect every
	// create operation can be atomic
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.controller.addAnnotationForCurrentPod(k, v); err != nil {
		return perrors.WithMessagef(err, "add annotation @key = %s @value = %s", k, v)
	}

	logger.Debugf("put the @key = %s @value = %s success", k, v)
	return nil
}

// GetChildren
// get k children list from kubernetes-watcherSet
func (c *Client) GetChildren(k string) ([]string, []string, error) {

	objectList, err := c.controller.watcherSet.Get(k, true)
	if err != nil {
		return nil, nil, perrors.WithMessagef(err, "get children from watcherSet on (%s)", k)
	}

	var kList []string
	var vList []string

	for _, o := range objectList {
		kList = append(kList, o.Key)
		vList = append(vList, o.Value)
	}

	return kList, vList, nil
}

// Watch
// watch on spec key
func (c *Client) Watch(k string) (<-chan *WatcherEvent, <-chan struct{}, error) {

	w, err := c.controller.watcherSet.Watch(k, false)
	if err != nil {
		return nil, nil, perrors.WithMessagef(err, "watch on (%s)", k)
	}

	return w.ResultChan(), w.done(), nil
}

// Watch
// watch on spec prefix
func (c *Client) WatchWithPrefix(prefix string) (<-chan *WatcherEvent, <-chan struct{}, error) {

	w, err := c.controller.watcherSet.Watch(prefix, true)
	if err != nil {
		return nil, nil, perrors.WithMessagef(err, "watch on prefix (%s)", prefix)
	}

	return w.ResultChan(), w.done(), nil
}

// Valid
// Valid the client
// if return false, the client is die
func (c *Client) Valid() bool {

	select {
	case <-c.Done():
		return false
	default:
	}
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.controller != nil
}

// Done
// read the client status
func (c *Client) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Stop
// read the client status
func (c *Client) Close() {

	select {
	case <-c.ctx.Done():
		//already stopped
		return
	default:
	}
	c.cancel()

	// the client ctx be canceled
	// will trigger the watcherSet watchers all stopped
	// so, just wait
}

// ValidateClient
// validate the kubernetes client
func ValidateClient(container clientFacade) error {

	client := container.Client()

	// new Client
	if client == nil || client.Valid() {
		//ns, err := getCurrentNameSpace()
		//if err != nil {
		//	return perrors.WithMessage(err, "get current namespace")
		//}
		//newClient, err := newClient(ns)
		newClient, err := newClient("")
		if err != nil {
			logger.Warnf("new kubernetes client (namespace{%s}: %v)", "", err)
			return perrors.WithMessagef(err, "new kubernetes client (:%+v)", "")
		}
		container.SetClient(newClient)
	}

	return nil
}

// NewMockClient
// export for registry package test
func NewMockClient(namespace string, mockClientGenerator func() (kubernetes.Interface, error)) (*Client, error) {
	return nil, nil
	//return newMockClient(namespace, mockClientGenerator)
}

// newMockClient
// new a client for  test
//func newMockClient(namespace string, mockClientGenerator func() (kubernetes.Interface, error)) (*Client, error) {
//
//	rawClient, err := mockClientGenerator()
//	if err != nil {
//		return nil, perrors.WithMessage(err, "call mock generator")
//	}
//
//	currentPodName, err := getCurrentPodName()
//	if err != nil {
//		return nil, perrors.WithMessage(err, "get pod name")
//	}
//
//	ctx, cancel := context.WithCancel(context.Background())
//
//	c := &Client{
//		currentPodName: currentPodName,
//		ns:             namespace,
//		rawClient:      rawClient,
//		ctx:            ctx,
//		watcherSet:     newWatcherSet(ctx),
//		cancel:         cancel,
//	}
//
//	currentPod, err := c.initCurrentPod()
//	if err != nil {
//		return nil, perrors.WithMessage(err, "init current pod")
//	}
//
//	// record current status
//	c.currentPod = currentPod
//
//	// init the watcherSet by current pods
//	if err := c.initWatchSet(); err != nil {
//		return nil, perrors.WithMessage(err, "init watcherSet")
//	}
//
//	c.lastResourceVersion = c.currentPod.GetResourceVersion()
//
//	// start kubernetes watch loop
//	if err := c.watchPods(); err != nil {
//		return nil, perrors.WithMessage(err, "watch pods")
//	}
//
//	logger.Infof("init kubernetes registry client success @namespace = %q @Podname = %q", namespace, c.currentPod.Name)
//	return c, nil
//}
