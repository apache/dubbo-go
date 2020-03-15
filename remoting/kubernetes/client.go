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
	"encoding/base64"
	"encoding/json"
	"os"
	"sync"
	"time"
)

import (
	perrors "github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

import (
	"github.com/apache/dubbo-go/common/logger"
)

const (
	// kubernetes inject the var
	podNameKey   = "HOSTNAME"
	nameSpaceKey = "NAMESPACE"
	// all pod annotation key
	DubboIOAnnotationKey = "dubbo.io/annotation"

	DubboIOLabelKey   = "dubbo.io/label"
	DubboIOLabelValue = "dubbo.io-value"
)

var (
	ErrDubboLabelAlreadyExist = perrors.New("dubbo label already exist")
)

type Client struct {

	// kubernetes connection config
	cfg *rest.Config

	// the kubernetes interface
	rawClient kubernetes.Interface

	// current pod config
	currentPodName string

	ns string

	// the memory watcherSet
	watcherSet WatcherSet

	// protect the wg && currentPod
	lock sync.RWMutex
	// current pod status
	currentPod *v1.Pod
	// protect the watchPods loop && watcher
	wg sync.WaitGroup

	// manage the  client lifecycle
	ctx    context.Context
	cancel context.CancelFunc
}

// load CurrentPodName
func getCurrentPodName() (string, error) {

	v := os.Getenv(podNameKey)
	if len(v) == 0 {
		return "", perrors.New("read value from env by key (HOSTNAME)")
	}
	return v, nil
}

// load CurrentNameSpace
func getCurrentNameSpace() (string, error) {

	v := os.Getenv(nameSpaceKey)
	if len(v) == 0 {
		return "", perrors.New("read value from env by key (NAMESPACE)")
	}
	return v, nil
}

// NewMockClient
// export for registry package test
func NewMockClient(namespace string, mockClientGenerator func() (kubernetes.Interface, error)) (*Client, error) {
	return newMockClient(namespace, mockClientGenerator)
}

// newMockClient
// new a client for  test
func newMockClient(namespace string, mockClientGenerator func() (kubernetes.Interface, error)) (*Client, error) {

	rawClient, err := mockClientGenerator()
	if err != nil {
		return nil, perrors.WithMessage(err, "call mock generator")
	}

	currentPodName, err := getCurrentPodName()
	if err != nil {
		return nil, perrors.WithMessage(err, "get pod name")
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Client{
		currentPodName: currentPodName,
		ns:             namespace,
		rawClient:      rawClient,
		ctx:            ctx,
		watcherSet:     newWatcherSet(ctx),
		cancel:         cancel,
	}

	currentPod, err := c.initCurrentPod()
	if err != nil {
		return nil, perrors.WithMessage(err, "init current pod")
	}

	// record current status
	c.currentPod = currentPod

	// init the watcherSet by current pods
	if err := c.initWatchSet(); err != nil {
		return nil, perrors.WithMessage(err, "init watcherSet")
	}

	// start kubernetes watch loop
	if err := c.watchPods(); err != nil {
		return nil, perrors.WithMessage(err, "maintenance the kubernetes status")
	}

	logger.Info("init kubernetes registry success")
	return c, nil
}

// newClient
// new a client for registry
func newClient(namespace string) (*Client, error) {

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, perrors.WithMessage(err, "get in-cluster config")
	}

	rawClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, perrors.WithMessage(err, "new kubernetes client by in cluster config")
	}

	currentPodName, err := getCurrentPodName()
	if err != nil {
		return nil, perrors.WithMessage(err, "get pod name")
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Client{
		currentPodName: currentPodName,
		ns:             namespace,
		cfg:            cfg,
		rawClient:      rawClient,
		ctx:            ctx,
		watcherSet:     newWatcherSet(ctx),
		cancel:         cancel,
	}

	currentPod, err := c.initCurrentPod()
	if err != nil {
		return nil, perrors.WithMessage(err, "init current pod")
	}

	// record current status
	c.currentPod = currentPod

	// init the watcherSet by current pods
	if err := c.initWatchSet(); err != nil {
		return nil, perrors.WithMessage(err, "init watcherSet")
	}

	// start kubernetes watch loop
	if err := c.watchPods(); err != nil {
		return nil, perrors.WithMessage(err, "maintenance the kubernetes status")
	}

	logger.Info("init kubernetes registry success")
	return c, nil
}

// initCurrentPod
// 1. get current pod
// 2. give the dubbo-label for this pod
func (c *Client) initCurrentPod() (*v1.Pod, error) {

	// read the current pod status
	currentPod, err := c.rawClient.CoreV1().Pods(c.ns).Get(c.currentPodName, metav1.GetOptions{})
	if err != nil {
		return nil, perrors.WithMessagef(err, "get current (%s) pod in namespace (%s)", c.currentPodName, c.ns)
	}

	oldPod, newPod, err := c.assembleDUBBOLabel(currentPod)
	if err != nil {
		if err != ErrDubboLabelAlreadyExist {
			return nil, perrors.WithMessage(err, "assemble dubbo label")
		}
		// current pod don't have label
	}

	p, err := c.getPatch(oldPod, newPod)
	if err != nil {
		return nil, perrors.WithMessage(err, "get patch")
	}

	currentPod, err = c.patchCurrentPod(p)
	if err != nil {
		return nil, perrors.WithMessage(err, "patch to current pod")
	}

	return currentPod, nil
}

// initWatchSet
// 1. get all with dubbo label pods
// 2. put every element to watcherSet
func (c *Client) initWatchSet() error {

	pods, err := c.rawClient.CoreV1().Pods(c.ns).List(metav1.ListOptions{
		LabelSelector: fields.OneTermEqualSelector(DubboIOLabelKey, DubboIOLabelValue).String(),
	})
	if err != nil {
		return perrors.WithMessagef(err, "list pods  in namespace (%s)", c.ns)
	}

	for _, pod := range pods.Items {
		logger.Debugf("got the pod (name: %s), (label: %v), (annotations: %v)", pod.Name, pod.GetLabels(), pod.GetAnnotations())
		c.handleWatchedPodEvent(&pod, watch.Added)
	}

	return nil
}

// watchPods
// try to watch kubernetes pods
func (c *Client) watchPods() error {

	// try once
	watcher, err := c.rawClient.CoreV1().Pods(c.ns).Watch(metav1.ListOptions{
		LabelSelector: fields.OneTermEqualSelector(DubboIOLabelKey, DubboIOLabelValue).String(),
		Watch:         true,
	})
	if err != nil {
		return perrors.WithMessagef(err, "try to watch the namespace (%s) pods", c.ns)
	}

	watcher.Stop()

	c.wg.Add(1)
	// add wg, grace close the client
	go c.watchPodsLoop()
	return nil
}

// watchPods
// try to notify
func (c *Client) watchPodsLoop() {

	defer func() {
		// notify other goroutine, this loop over
		c.wg.Done()
		logger.Info("watchPodsLoop goroutine game over")
	}()

	var lastResourceVersion string

	for {

		wc, err := c.rawClient.CoreV1().Pods(c.ns).Watch(metav1.ListOptions{
			LabelSelector:   fields.OneTermEqualSelector(DubboIOLabelKey, DubboIOLabelValue).String(),
			Watch:           true,
			ResourceVersion: lastResourceVersion,
		})
		if err != nil {
			logger.Warnf("watch the namespace (%s) pods: %v, retry after 2 seconds", c.ns, err)
			time.Sleep(2 * time.Second)
			continue
		}

		logger.Infof("the old kubernetes client broken, collect the resource status from resource version (%s)", lastResourceVersion)

		select {
		case <-c.ctx.Done():
			// the client stopped
			logger.Info("the kubernetes client stopped")
			return

		default:

			for {
				select {
				// double check ctx
				case <-c.ctx.Done():
					logger.Info("the kubernetes client stopped")
					goto onceWatch

					// get one element from result-chan
				case event, ok := <-wc.ResultChan():
					if !ok {
						wc.Stop()
						logger.Info("kubernetes watch chan die, create new")
						goto onceWatch
					}

					if event.Type == watch.Error {
						// watched a error event
						logger.Warnf("kubernetes watch api report err (%#v)", event)
						continue
					}

					type resourceVersionGetter interface {
						GetResourceVersion() string
					}

					o, ok := event.Object.(resourceVersionGetter)
					if !ok {
						continue
					}

					// record the last resource version avoid to sync all pod
					lastResourceVersion = o.GetResourceVersion()
					logger.Infof("kuberentes get the current resource version %v", lastResourceVersion)

					// check event object type
					p, ok := event.Object.(*v1.Pod)
					if !ok {
						// not a pod
						continue
					}

					// handle the watched pod
					go c.handleWatchedPodEvent(p, event.Type)
				}
			}
		onceWatch:
		}
	}
}

// handleWatchedPodEvent
// handle watched pod event
func (c *Client) handleWatchedPodEvent(p *v1.Pod, eventType watch.EventType) {

	for ak, av := range p.GetAnnotations() {

		// not dubbo interest annotation
		if ak != DubboIOAnnotationKey {
			continue
		}

		ol, err := c.unmarshalRecord(av)
		if err != nil {
			logger.Errorf("there a pod with dubbo annotation, but unmarshal dubbo value %v", err)
			return
		}

		for _, o := range ol {

			switch eventType {
			case watch.Added:
				// if pod is added, the record always be create
				o.EventType = Create
			case watch.Modified:
				o.EventType = Update
			case watch.Deleted:
				o.EventType = Delete
			default:
				logger.Errorf("no valid kubernetes event-type (%s) ", eventType)
				return
			}

			logger.Debugf("prepare to put object (%#v) to kuberentes-watcherSet", o)

			if err := c.watcherSet.Put(o); err != nil {
				logger.Errorf("put (%#v) to cache watcherSet: %v ", o, err)
				return
			}

		}

	}
}

// unmarshalRecord
// unmarshal the kubernetes dubbo annotation value
func (c *Client) unmarshalRecord(record string) ([]*WatcherEvent, error) {

	if len(record) == 0 {
		// []*WatcherEvent is nil.
		return nil, nil
	}

	rawMsg, err := base64.URLEncoding.DecodeString(record)
	if err != nil {
		return nil, perrors.WithMessagef(err, "decode record (%s)", record)
	}

	var out []*WatcherEvent
	if err := json.Unmarshal(rawMsg, &out); err != nil {
		return nil, perrors.WithMessage(err, "decode json")
	}
	return out, nil
}

// marshalRecord
// marshal the kubernetes dubbo annotation value
func (c *Client) marshalRecord(ol []*WatcherEvent) (string, error) {

	msg, err := json.Marshal(ol)
	if err != nil {
		return "", perrors.WithMessage(err, "json encode object list")
	}
	return base64.URLEncoding.EncodeToString(msg), nil
}

// readCurrentPod
// read the current pod status from kubernetes api
func (c *Client) readCurrentPod() (*v1.Pod, error) {

	currentPod, err := c.rawClient.CoreV1().Pods(c.ns).Get(c.currentPodName, metav1.GetOptions{})
	if err != nil {
		return nil, perrors.WithMessagef(err, "get current (%s) pod in namespace (%s)", c.currentPodName, c.ns)
	}
	return currentPod, nil
}

// Create
// create k/v pair in storage
func (c *Client) Create(k, v string) error {

	// 1. accord old pod && (k, v) assemble new pod dubbo annotion v
	// 2. get patch data
	// 3. PATCH the pod
	c.lock.Lock()
	defer c.lock.Unlock()

	currentPod, err := c.readCurrentPod()
	if err != nil {
		return perrors.WithMessage(err, "read current pod")
	}

	oldPod, newPod, err := c.assembleDUBBOAnnotations(k, v, currentPod)
	if err != nil {
		return perrors.WithMessage(err, "assemble")
	}

	patchBytes, err := c.getPatch(oldPod, newPod)
	if err != nil {
		return perrors.WithMessage(err, "get patch")
	}

	updatedPod, err := c.patchCurrentPod(patchBytes)
	if err != nil {
		return perrors.WithMessage(err, "patch current pod")
	}

	c.currentPod = updatedPod
	// not update the watcherSet, the watcherSet should be write by the  watchPodsLoop
	return nil
}

// patch current pod
// write new meta for current pod
func (c *Client) patchCurrentPod(patch []byte) (*v1.Pod, error) {

	updatedPod, err := c.rawClient.CoreV1().Pods(c.ns).Patch(c.currentPodName, types.StrategicMergePatchType, patch)
	if err != nil {
		return nil, perrors.WithMessage(err, "patch in kubernetes pod ")
	}
	return updatedPod, nil
}

// assemble the dubbo kubernetes label
// every dubbo instance should be labeled spec {"dubbo.io/label":"dubbo.io/label-value"} label
func (c *Client) assembleDUBBOLabel(currentPod *v1.Pod) (*v1.Pod, *v1.Pod, error) {

	var (
		oldPod = &v1.Pod{}
		newPod = &v1.Pod{}
	)

	oldPod.Labels = make(map[string]string, 8)
	newPod.Labels = make(map[string]string, 8)

	if currentPod.GetLabels() != nil {

		if currentPod.GetLabels()[DubboIOLabelKey] == DubboIOLabelValue {
			// already have label
			return nil, nil, ErrDubboLabelAlreadyExist
		}
	}

	// copy current pod labels to oldPod && newPod
	for k, v := range currentPod.GetLabels() {
		oldPod.Labels[k] = v
		newPod.Labels[k] = v
	}
	// assign new label for current pod
	newPod.Labels[DubboIOLabelKey] = DubboIOLabelValue
	return oldPod, newPod, nil
}

// assemble the dubbo kubernetes annotations
// accord the current pod && (k,v) assemble the old-pod, new-pod
func (c *Client) assembleDUBBOAnnotations(k, v string, currentPod *v1.Pod) (oldPod *v1.Pod, newPod *v1.Pod, err error) {

	oldPod = &v1.Pod{}
	newPod = &v1.Pod{}
	oldPod.Annotations = make(map[string]string, 8)
	newPod.Annotations = make(map[string]string, 8)

	for k, v := range currentPod.GetAnnotations() {
		oldPod.Annotations[k] = v
		newPod.Annotations[k] = v
	}

	al, err := c.unmarshalRecord(oldPod.GetAnnotations()[DubboIOAnnotationKey])
	if err != nil {
		err = perrors.WithMessage(err, "unmarshal record")
		return
	}

	newAnnotations, err := c.marshalRecord(append(al, &WatcherEvent{Key: k, Value: v}))
	if err != nil {
		err = perrors.WithMessage(err, "marshal record")
		return
	}

	newPod.Annotations[DubboIOAnnotationKey] = newAnnotations
	return
}

// getPatch
// get the kubernetes pod patch bytes
func (c *Client) getPatch(oldPod, newPod *v1.Pod) ([]byte, error) {

	oldData, err := json.Marshal(oldPod)
	if err != nil {
		return nil, perrors.WithMessage(err, "marshal old pod")
	}

	newData, err := json.Marshal(newPod)
	if err != nil {
		return nil, perrors.WithMessage(err, "marshal newPod pod")
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, v1.Pod{})
	if err != nil {
		return nil, perrors.WithMessage(err, "create two-way-merge-patch")
	}
	return patchBytes, nil
}

// GetChildren
// get k children list from kubernetes-watcherSet
func (c *Client) GetChildren(k string) ([]string, []string, error) {

	objectList, err := c.watcherSet.Get(k, true)
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

	w, err := c.watcherSet.Watch(k, false)
	if err != nil {
		return nil, nil, perrors.WithMessagef(err, "watch on (%s)", k)
	}

	return w.ResultChan(), w.done(), nil
}

// Watch
// watch on spec prefix
func (c *Client) WatchWithPrefix(prefix string) (<-chan *WatcherEvent, <-chan struct{}, error) {

	w, err := c.watcherSet.Watch(prefix, true)
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
	if c.rawClient == nil {
		c.lock.RUnlock()
		return false
	}
	c.lock.RUnlock()
	return true
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
	c.wg.Wait()
}

// ValidateClient
// validate the kubernetes client
func ValidateClient(container clientFacade) error {

	lock := container.ClientLock()
	lock.Lock()
	defer lock.Unlock()

	// new Client
	if container.Client() == nil {
		ns, err := getCurrentNameSpace()
		if err != nil {
			return perrors.WithMessage(err, "get current namespace")
		}
		newClient, err := newClient(ns)
		if err != nil {
			logger.Warnf("new kubernetes client (namespace{%s}: %v)", ns, err)
			return perrors.WithMessagef(err, "new kubernetes client (:%+v)", ns)
		}
		container.SetClient(newClient)
	}

	if !container.Client().Valid() {

		ns, err := getCurrentNameSpace()
		if err != nil {
			return perrors.WithMessage(err, "get current namespace")
		}
		newClient, err := newClient(ns)
		if err != nil {
			logger.Warnf("new kubernetes client (namespace{%s}: %v)", ns, err)
			return perrors.WithMessagef(err, "new kubernetes client (:%+v)", ns)
		}
		container.SetClient(newClient)
	}

	return nil
}
