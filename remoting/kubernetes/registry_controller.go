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
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	perrors "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	informerscorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
)

const (
	// kubernetes inject env var
	podNameKey              = "HOSTNAME"
	nameSpaceKey            = "NAMESPACE"
	needWatchedNameSpaceKey = "DUBBO_NAMESPACE"

	// all pod annotation key
	DubboIOAnnotationKey = "dubbo.io/annotation"
	// all pod label key and value pair
	DubboIOLabelKey           = "dubbo.io/label"
	DubboIOConsumerLabelValue = "dubbo.io.consumer"
	DubboIOProviderLabelValue = "dubbo.io.provider"

	// kubernetes suggest resync
	defaultResync = 5 * time.Minute
)

var ErrDubboLabelAlreadyExist = perrors.New("dubbo label already exist")

// dubboRegistryController works like a kubernetes controller
type dubboRegistryController struct {

	// clone from client
	// manage lifecycle
	ctx context.Context

	role common.RoleType

	// protect patch current pod operation
	lock sync.Mutex

	// current pod config
	needWatchedNamespace map[string]struct{}
	namespace            string
	name                 string

	watcherSet WatcherSet

	// kubernetes
	kc                               kubernetes.Interface
	listAndWatchStartResourceVersion uint64
	namespacedInformerFactory        map[string]informers.SharedInformerFactory
	namespacedPodInformers           map[string]informerscorev1.PodInformer
	queue                            workqueue.Interface // shared by namespaced informers
}

func newDubboRegistryController(
	ctx context.Context,
	// different provider and consumer have behavior
	roleType common.RoleType,
	// used to inject mock kubernetes client
	kcGetter func() (kubernetes.Interface, error),
) (*dubboRegistryController, error) {

	kc, err := kcGetter()
	if err != nil {
		return nil, perrors.WithMessage(err, "get kubernetes client")
	}

	c := &dubboRegistryController{
		ctx:                       ctx,
		role:                      roleType,
		watcherSet:                newWatcherSet(ctx),
		needWatchedNamespace:      make(map[string]struct{}),
		namespacedInformerFactory: make(map[string]informers.SharedInformerFactory),
		namespacedPodInformers:    make(map[string]informerscorev1.PodInformer),
		kc:                        kc,
	}

	if err := c.readConfig(); err != nil {
		return nil, perrors.WithMessage(err, "read config")
	}

	if err := c.initCurrentPod(); err != nil {
		return nil, perrors.WithMessage(err, "init current pod")
	}

	if err := c.initWatchSet(); err != nil {
		return nil, perrors.WithMessage(err, "init watch set")
	}

	if err := c.initPodInformer(); err != nil {
		return nil, perrors.WithMessage(err, "init pod informer")
	}

	go c.run()

	return c, nil
}

// GetInClusterKubernetesClient
// current pod running in kubernetes-cluster
func GetInClusterKubernetesClient() (kubernetes.Interface, error) {
	// read in-cluster config
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, perrors.WithMessage(err, "get in-cluster config")
	}

	return kubernetes.NewForConfig(cfg)
}

// initWatchSet
// 1. get all with dubbo label pods
// 2. put every element to watcherSet
// 3. refresh watch book-mark
func (c *dubboRegistryController) initWatchSet() error {
	req, err := labels.NewRequirement(DubboIOLabelKey, selection.In, []string{DubboIOConsumerLabelValue, DubboIOProviderLabelValue})
	if err != nil {
		return perrors.WithMessage(err, "new requirement")
	}

	for ns := range c.needWatchedNamespace {
		pods, err := c.kc.CoreV1().Pods(ns).List(metav1.ListOptions{
			LabelSelector: req.String(),
		})
		if err != nil {
			return perrors.WithMessagef(err, "list pods in namespace (%s)", ns)
		}
		for _, p := range pods.Items {
			// set resource version
			rv, err := strconv.ParseUint(p.GetResourceVersion(), 10, 0)
			if err != nil {
				return perrors.WithMessagef(err, "parse resource version %s", p.GetResourceVersion())
			}
			if c.listAndWatchStartResourceVersion < rv {
				c.listAndWatchStartResourceVersion = rv
			}
			c.handleWatchedPodEvent(&p, watch.Added)
		}
	}
	return nil
}

// read dubbo-registry controller config
// 1. current pod name
// 2. current pod working namespace
func (c *dubboRegistryController) readConfig() error {
	// read current pod name && namespace
	c.name = os.Getenv(podNameKey)
	if len(c.name) == 0 {
		return perrors.New("read value from env by key (HOSTNAME)")
	}
	c.namespace = os.Getenv(nameSpaceKey)
	if len(c.namespace) == 0 {
		return perrors.New("read value from env by key (NAMESPACE)")
	}
	return nil
}

func (c *dubboRegistryController) initNamespacedPodInformer(ns string) error {
	req, err := labels.NewRequirement(DubboIOLabelKey, selection.In, []string{DubboIOConsumerLabelValue, DubboIOProviderLabelValue})
	if err != nil {
		return perrors.WithMessage(err, "new requirement")
	}

	informersFactory := informers.NewSharedInformerFactoryWithOptions(
		c.kc,
		defaultResync,
		informers.WithNamespace(ns),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.LabelSelector = req.String()
			options.ResourceVersion = strconv.FormatUint(c.listAndWatchStartResourceVersion, 10)
		}),
	)
	podInformer := informersFactory.Core().V1().Pods()

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addPod,
		UpdateFunc: c.updatePod,
		DeleteFunc: c.deletePod,
	})

	c.namespacedInformerFactory[ns] = informersFactory
	c.namespacedPodInformers[ns] = podInformer

	return nil
}

func (c *dubboRegistryController) initPodInformer() error {
	if c.role == common.PROVIDER {
		return nil
	}

	// read need watched namespaces list
	needWatchedNameSpaceList := os.Getenv(needWatchedNameSpaceKey)
	if len(needWatchedNameSpaceList) == 0 {
		return perrors.New("read value from env by key (DUBBO_NAMESPACE)")
	}
	for _, ns := range strings.Split(needWatchedNameSpaceList, ",") {
		c.needWatchedNamespace[ns] = struct{}{}
	}
	// current work namespace should be watched
	c.needWatchedNamespace[c.namespace] = struct{}{}

	c.queue = workqueue.New()

	// init all watch needed pod-informer
	for watchedNS := range c.needWatchedNamespace {
		if err := c.initNamespacedPodInformer(watchedNS); err != nil {
			return err
		}
	}
	return nil
}

type kubernetesEvent struct {
	p *v1.Pod
	t watch.EventType
}

func (c *dubboRegistryController) addPod(obj interface{}) {
	p, ok := obj.(*v1.Pod)
	if !ok {
		logger.Warnf("pod-informer got object %T not *v1.Pod", obj)
		return
	}
	c.queue.Add(&kubernetesEvent{
		t: watch.Added,
		p: p,
	})
}

func (c *dubboRegistryController) updatePod(oldObj, newObj interface{}) {
	op, ok := oldObj.(*v1.Pod)
	if !ok {
		logger.Warnf("pod-informer got object %T not *v1.Pod", oldObj)
		return
	}
	np, ok := newObj.(*v1.Pod)
	if !ok {
		logger.Warnf("pod-informer got object %T not *v1.Pod", newObj)
		return
	}
	if op.GetResourceVersion() == np.GetResourceVersion() {
		return
	}
	c.queue.Add(&kubernetesEvent{
		p: np,
		t: watch.Modified,
	})
}

func (c *dubboRegistryController) deletePod(obj interface{}) {
	p, ok := obj.(*v1.Pod)
	if !ok {
		logger.Warnf("pod-informer got object %T not *v1.Pod", obj)
		return
	}
	c.queue.Add(&kubernetesEvent{
		p: p,
		t: watch.Deleted,
	})
}

func (c *dubboRegistryController) startALLInformers() {
	logger.Debugf("starting namespaced informer-factory")
	for _, factory := range c.namespacedInformerFactory {
		go factory.Start(c.ctx.Done())
	}
}

// run
// controller process every event in work-queue
func (c *dubboRegistryController) run() {
	if c.role == common.PROVIDER {
		return
	}

	defer logger.Warn("dubbo registry controller work stopped")
	defer c.queue.ShutDown()

	for ns, podInformer := range c.namespacedPodInformers {
		if !cache.WaitForCacheSync(c.ctx.Done(), podInformer.Informer().HasSynced) {
			logger.Errorf("wait for cache sync finish @namespace %s fail", ns)
			return
		}
	}

	logger.Infof("kubernetes registry-controller running @Namespace = %q @PodName = %q", c.namespace, c.name)

	// start work
	go c.work()
	// block wait context cancel
	<-c.ctx.Done()
}

func (c *dubboRegistryController) work() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem process work-queue elements
func (c *dubboRegistryController) processNextWorkItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(item)
	o := item.(*kubernetesEvent)
	c.handleWatchedPodEvent(o.p, o.t)
	return true
}

// handleWatchedPodEvent handles watched pod event
func (c *dubboRegistryController) handleWatchedPodEvent(p *v1.Pod, eventType watch.EventType) {
	logger.Debugf("get @type = %s event from @pod = %s", eventType, p.GetName())

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

			logger.Debugf("putting @key=%s @value=%s to watcherSet", o.Key, o.Value)
			if err := c.watcherSet.Put(o); err != nil {
				logger.Errorf("put (%#v) to cache watcherSet: %v ", o, err)
				return
			}
		}
	}
}

// unmarshalRecord unmarshals the kubernetes dubbo annotation value
func (c *dubboRegistryController) unmarshalRecord(record string) ([]*WatcherEvent, error) {
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

// initCurrentPod
// 1. get current pod
// 2. give the dubbo-label for this pod
func (c *dubboRegistryController) initCurrentPod() error {
	currentPod, err := c.readCurrentPod()
	if err != nil {
		return perrors.WithMessagef(err, "get current (%s) pod in namespace (%s)", c.name, c.namespace)
	}

	oldPod, newPod, err := c.assembleDUBBOLabel(currentPod)
	if err != nil {
		if err == ErrDubboLabelAlreadyExist {
			return nil
		}
		return perrors.WithMessage(err, "assemble dubbo label")
	}
	// current pod don't have label
	p, err := c.getPatch(oldPod, newPod)
	if err != nil {
		return perrors.WithMessage(err, "get patch")
	}

	_, err = c.patchCurrentPod(p)
	if err != nil {
		return perrors.WithMessage(err, "patch to current pod")
	}

	return nil
}

// patchCurrentPod writes new meta for current pod
func (c *dubboRegistryController) patchCurrentPod(patch []byte) (*v1.Pod, error) {
	updatedPod, err := c.kc.CoreV1().Pods(c.namespace).Patch(c.name, types.StrategicMergePatchType, patch)
	if err != nil {
		return nil, perrors.WithMessage(err, "patch in kubernetes pod ")
	}
	return updatedPod, nil
}

// assembleDUBBOLabel assembles the dubbo kubernetes label
// every dubbo instance should be labeled spec {"dubbo.io/label":"dubbo.io/label-value"} label
func (c *dubboRegistryController) assembleDUBBOLabel(p *v1.Pod) (*v1.Pod, *v1.Pod, error) {
	var (
		oldPod = &v1.Pod{}
		newPod = &v1.Pod{}
	)
	oldPod.Labels = make(map[string]string, 8)
	newPod.Labels = make(map[string]string, 8)

	if p.GetLabels() != nil {
		if _, ok := p.GetLabels()[DubboIOLabelKey]; ok {
			// already have label
			return nil, nil, ErrDubboLabelAlreadyExist
		}
	}

	// copy current pod labels to oldPod && newPod
	for k, v := range p.GetLabels() {
		oldPod.Labels[k] = v
		newPod.Labels[k] = v
	}

	// assign new label for current pod
	switch c.role {
	case common.CONSUMER:
		newPod.Labels[DubboIOLabelKey] = DubboIOConsumerLabelValue
	case common.PROVIDER:
		newPod.Labels[DubboIOLabelKey] = DubboIOProviderLabelValue
	default:
		return nil, nil, perrors.New(fmt.Sprintf("unknown role %s", c.role))
	}
	return oldPod, newPod, nil
}

// assembleDUBBOAnnotations assembles the dubbo kubernetes annotations
// accord the current pod && (k,v) assemble the old-pod, new-pod
func (c *dubboRegistryController) assembleDUBBOAnnotations(k, v string, currentPod *v1.Pod) (oldPod *v1.Pod, newPod *v1.Pod, err error) {
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

// getPatch gets the kubernetes pod patch bytes
func (c *dubboRegistryController) getPatch(oldPod, newPod *v1.Pod) ([]byte, error) {
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

// marshalRecord marshals the kubernetes dubbo annotation value
func (c *dubboRegistryController) marshalRecord(ol []*WatcherEvent) (string, error) {
	msg, err := json.Marshal(ol)
	if err != nil {
		return "", perrors.WithMessage(err, "json encode object list")
	}
	return base64.URLEncoding.EncodeToString(msg), nil
}

// readCurrentPod reads from kubernetes-env current pod status
func (c *dubboRegistryController) readCurrentPod() (*v1.Pod, error) {
	currentPod, err := c.kc.CoreV1().Pods(c.namespace).Get(c.name, metav1.GetOptions{})
	if err != nil {
		return nil, perrors.WithMessagef(err, "get current (%s) pod in namespace (%s)", c.name, c.namespace)
	}
	return currentPod, nil
}

// addAnnotationForCurrentPod adds annotation for current pod
func (c *dubboRegistryController) addAnnotationForCurrentPod(k string, v string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 1. accord old pod && (k, v) assemble new pod dubbo annotation v
	// 2. get patch data
	// 3. PATCH the pod
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

	_, err = c.patchCurrentPod(patchBytes)
	if err != nil {
		return perrors.WithMessage(err, "patch current pod")
	}

	return c.watcherSet.Put(&WatcherEvent{
		Key:       k,
		Value:     v,
		EventType: Create,
	})
}
