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
	"strconv"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

type Client struct {
	lock sync.RWMutex

	// manage the  client lifecycle
	ctx    context.Context
	cancel context.CancelFunc

	controller *dubboRegistryController
}

// newClient returns Client instance for registry
func newClient(url *common.URL) (*Client, error) {

	ctx, cancel := context.WithCancel(context.Background())

	// read type
	r, err := strconv.Atoi(url.GetParams().Get(constant.ROLE_KEY))
	if err != nil {
		return nil, perrors.WithMessage(err, "atoi role")
	}
	controller, err := newDubboRegistryController(ctx, common.RoleType(r), GetInClusterKubernetesClient)
	if err != nil {
		return nil, perrors.WithMessage(err, "new dubbo-registry controller")
	}

	c := &Client{
		ctx:        ctx,
		cancel:     cancel,
		controller: controller,
	}

	if r == common.CONSUMER {
		// only consumer have to start informer factory
		c.controller.startALLInformers()
	}
	return c, nil
}

// Create creates k/v pair in watcher-set
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

// GetChildren gets k children list from kubernetes-watcherSet
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

// Watch watches on spec key
func (c *Client) Watch(k string) (<-chan *WatcherEvent, <-chan struct{}, error) {

	w, err := c.controller.watcherSet.Watch(k, false)
	if err != nil {
		return nil, nil, perrors.WithMessagef(err, "watch on (%s)", k)
	}

	return w.ResultChan(), w.done(), nil
}

// WatchWithPrefix watches on spec prefix
func (c *Client) WatchWithPrefix(prefix string) (<-chan *WatcherEvent, <-chan struct{}, error) {

	w, err := c.controller.watcherSet.Watch(prefix, true)
	if err != nil {
		return nil, nil, perrors.WithMessagef(err, "watch on prefix (%s)", prefix)
	}

	return w.ResultChan(), w.done(), nil
}

// if returns false, the client is die
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

// nolint
func (c *Client) Done() <-chan struct{} {
	return c.ctx.Done()
}

// nolint
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

// ValidateClient validates the kubernetes client
func ValidateClient(container clientFacade) error {

	client := container.Client()

	// new Client
	if client == nil || client.Valid() {

		newClient, err := newClient(container.GetUrl())
		if err != nil {
			logger.Warnf("new kubernetes client: %v)", err)
			return perrors.WithMessage(err, "new kubernetes client")
		}
		container.SetClient(newClient)
	}

	return nil
}

// NewMockClient exports for registry package test
func NewMockClient(podList *v1.PodList) (*Client, error) {

	ctx, cancel := context.WithCancel(context.Background())
	controller, err := newDubboRegistryController(ctx, common.CONSUMER, func() (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(podList), nil
	})
	if err != nil {
		return nil, perrors.WithMessage(err, "new dubbo-registry controller")
	}

	c := &Client{
		ctx:        ctx,
		cancel:     cancel,
		controller: controller,
	}

	c.controller.startALLInformers()
	return c, nil
}
