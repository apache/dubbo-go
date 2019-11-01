package kubernetes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"sync"
)
import (
	perrors "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	DubboAnnotationKey = "DUBBO"
)

type Client struct {

	// kubernetes connection config
	cfg *rest.Config

	// the kubernetes interface
	rawClient kubernetes.Interface

	// current pod config
	currentPodName string

	ns string

	// the memory store
	store Store

	// protect the wg && currentPod
	lock sync.Mutex
	// current pod status
	currentPod *v1.Pod
	// protect the maintenanceStatus loop && watcher
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
		store:          newStore(ctx),
		cancel:         cancel,
	}

	// read the current pod status
	currentPods, err := rawClient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, perrors.WithMessagef(err, "list pods  in namespace (%s)", namespace)
	}

	// init the store by current pods
	c.initStore(currentPods)

	// start kubernetes watch loop
	if err := c.maintenanceStatus(); err != nil {
		return nil, perrors.WithMessage(err, "maintenance the kubernetes status")
	}

	logger.Info("init kubernetes registry success")
	return c, nil
}

// initStore
// init the store
func (c *Client) initStore(pods *v1.PodList) {
	for _, pod := range pods.Items {
		if pod.Name == c.currentPodName {
			c.currentPod = &pod
		}
		c.handleWatchedPodEvent(&pod, watch.Added)
	}
}

// maintenanceStatus
// try to watch kubernetes pods
func (c *Client) maintenanceStatus() error {

	wc, err := c.rawClient.CoreV1().Pods(c.ns).Watch(metav1.ListOptions{
		Watch: true,
	})
	if err != nil {
		return perrors.WithMessagef(err, "watch  the namespace (%s) pods", c.ns)
	}

	// add wg, grace close the client
	c.wg.Add(1)
	go c.maintenanceStatusLoop(wc)
	return nil
}

// maintenanceStatus
// try to notify
func (c *Client) maintenanceStatusLoop(wc watch.Interface) {

	defer func() {
		wc.Stop()
		// notify other goroutine, this loop over
		c.wg.Done()
		logger.Info("maintenanceStatusLoop goroutine game over")
	}()

	for {

		select {
		case <-c.ctx.Done():
			// the client stopped
			logger.Info("the kubernetes client stopped")
			return
		default:
			// get one element from result-chan
			event, ok := <-wc.ResultChan()
			if !ok {
				logger.Info("watch result chan be stopped")
				return
			}

			if event.Type == watch.Error {
				// watched a error event
				logger.Warnf("kubernetes watch api report err (%#v)", event)
				return
			}

			// check event object type
			p, ok := event.Object.(*v1.Pod)
			if !ok {
				// not a pod
				continue
			}

			go c.handleWatchedPodEvent(p, event.Type)
		}
	}
}

// handleWatchedPodEvent
// handle watched pod event
func (c *Client) handleWatchedPodEvent(p *v1.Pod, eventType watch.EventType) {

	for ak, av := range p.GetAnnotations() {

		// not dubbo interest annotation
		if ak != DubboAnnotationKey {
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

			if err := c.store.Put(o); err != nil {
				logger.Errorf("put (%#v) to cache store: %v ", o, err)
				return
			}

		}

	}
}

// unmarshalRecord
// unmarshal the kubernetes dubbo annotation value
func (c *Client) unmarshalRecord(record string) ([]*Object, error) {

	if len(record) == 0 {
		// NOTICE:
		// []*Object is nil.
		return nil, nil
	}

	rawMsg, err := base64.URLEncoding.DecodeString(record)
	if err != nil {
		return nil, perrors.WithMessagef(err, "decode record (%s)", record)
	}

	var out []*Object
	if err := json.Unmarshal(rawMsg, &out); err != nil {
		return nil, perrors.WithMessage(err, "decode json")
	}
	return out, nil
}

// marshalRecord
// marshal the kubernetes dubbo annotation value
func (c *Client) marshalRecord(ol []*Object) (string, error) {

	msg, err := json.Marshal(ol)
	if err != nil {
		return "", perrors.WithMessage(err, "json encode object list")
	}
	return base64.URLEncoding.EncodeToString(msg), nil
}

// Create
// create k/v pair in storage
func (c *Client) Create(k, v string) error {

	// 1. accord old pod && (k, v) assemble new pod dubbo annotion v
	// 2. get patch data
	// 3. PATCH the pod
	c.lock.Lock()
	defer c.lock.Unlock()

	oldPod, newPod, err := c.assemble(k, v)
	if err != nil {
		return perrors.WithMessage(err, "assemble")
	}

	patchBytes, err := c.getPatch(oldPod, newPod)
	if err != nil {
		return perrors.WithMessage(err, "get patch")
	}

	_, err = c.rawClient.CoreV1().Pods(c.ns).Patch(c.currentPodName, types.StrategicMergePatchType, patchBytes)
	if err != nil {
		return perrors.WithMessage(err, "patch in kubernetes pod ")
	}

	return nil
}

// assemble
// accord the current pod && (k,v) assemble the old-pod, new-pod
func (c *Client) assemble(k string, v string) (oldPod *v1.Pod, newPod *v1.Pod, err error) {

	currentPod, err := c.rawClient.CoreV1().Pods(c.ns).Get(c.currentPodName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, perrors.WithMessage(err, "get current pod from kubernetes")
	}

	// refresh currentPod
	c.currentPod = currentPod

	oldAnnotations := c.currentPod.GetAnnotations()
	var oldDubboAnnotation string
	for k, v := range oldAnnotations {
		if k == DubboAnnotationKey {
			oldDubboAnnotation = v
		}
	}

	ol, err := c.unmarshalRecord(oldDubboAnnotation)
	if err != nil {
		return nil, nil, perrors.WithMessage(err, "unmarshal dubbo record")
	}

	ol = append(ol, &Object{
		Key:   k,
		Value: v,
	})

	newV, err := c.marshalRecord(ol)
	if err != nil {
		return nil, nil, perrors.WithMessage(err, "marshal dubbo record")
	}

	newAnnotations := make(map[string]string)
	for k, v := range oldAnnotations {
		// set the old
		newAnnotations[k] = v
	}

	newAnnotations[DubboAnnotationKey] = newV
	newP := &v1.Pod{}
	newP.Annotations = newAnnotations
	newPod = newP

	oldP := &v1.Pod{}
	oldP.Annotations = oldAnnotations
	oldPod = oldP
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
// get k children list from kubernetes-store
func (c *Client) GetChildren(k string) ([]string, []string, error) {

	objectList, err := c.store.Get(k, true)
	if err != nil {
		return nil, nil, perrors.WithMessagef(err, "get children from store on (%s)", k)
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
func (c *Client) Watch(k string) (<-chan *Object, error) {

	w, err := c.store.Watch(k, false)
	if err != nil {
		return nil, perrors.WithMessagef(err, "watch on (%s)", k)
	}

	return w.ResultChan(), nil
}

// Watch
// watch on spec prefix
func (c *Client) WatchWithPrefix(prefix string) (<-chan *Object, error) {

	w, err := c.store.Watch(prefix, true)
	if err != nil {
		return nil, perrors.WithMessagef(err, "watch on prefix (%s)", prefix)
	}

	return w.ResultChan(), nil
}

// Valid
// Valid the client
// if return false, the client is die
func (c *Client) Valid() bool {

	select {
	case <-c.Done():
		return false
	default:
		return true
	}
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
	// will trigger the store watchers all stopped
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
