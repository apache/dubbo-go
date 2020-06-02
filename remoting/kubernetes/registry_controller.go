package kubernetes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"os"
	"sync"
	"time"

	"github.com/apache/dubbo-go/common/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	perrors "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	informerscorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/workqueue"
)

const (
	// kubernetes suggest resync
	defaultResync = 5 * time.Minute
)

// dubboRegistryController
// work like a kubernetes controller
type dubboRegistryController struct {

	// clone from client
	// manage lifecycle
	ctx context.Context

	// protect patch current pod operation
	lock sync.Mutex

	// current pod config
	needWatchedNamespaceList []string
	namespace                string
	name                     string
	cfg                      *rest.Config

	watcherSet WatcherSet

	// kubernetes
	kc                        kubernetes.Interface
	namespacedInformerFactory map[string]informers.SharedInformerFactory
	namespacedPodInformers    map[string]informerscorev1.PodInformer
	queue                     workqueue.Interface //shared by namespaced informers
}

func newDubboRegistryController(ctx context.Context) (*dubboRegistryController, error) {

	c := &dubboRegistryController{
		ctx:        ctx,
		watcherSet: newWatcherSet(ctx),
	}

	if err := c.readConfig(); err != nil {
		return nil, perrors.WithMessage(err, "dubbo registry controller read config")
	}

	if err := c.init(); err != nil {
		return nil,perrors.WithMessage(err, "dubbo registry controller init")
	}

	if _, err := c.initCurrentPod(); err != nil {
		return nil,perrors.WithMessage(err, "init current pod")
	}

	go c.run()
	return c, nil
}

// read dubbo-registry controller config
// 1. read kubernetes InCluster config
// 1. current pod name
// 2. current pod working namespace
func (c *dubboRegistryController) readConfig() error {
	// read in-cluster config
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return perrors.WithMessage(err, "get in-cluster config")
	}
	c.cfg = cfg

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

func (c *dubboRegistryController) initNamespacedPodInformer(ns string) {

	informersFactory := informers.NewSharedInformerFactoryWithOptions(
		c.kc,
		defaultResync,
		informers.WithNamespace(ns),
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			labelSelector := &metav1.LabelSelector{
				MatchLabels: map[string]string{DubboIOLabelKey: DubboIOLabelValue},
			}
			labelMap, err := metav1.LabelSelectorAsMap(labelSelector)
			if err != nil {
				panic(err)
			}
			options.LabelSelector = labels.SelectorFromSet(labelMap).String()
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
}

func (c *dubboRegistryController) init() error {
	// init kubernetes client
	var err error
	c.kc, err = kubernetes.NewForConfig(c.cfg)
	if err != nil {
		return perrors.WithMessage(err, "new kubernetes client from config")
	}
	c.queue = workqueue.New()

	c.needWatchedNamespaceList = append(c.needWatchedNamespaceList, c.namespace)

	// init all watch needed pod-informer
	for _, watchedNS := range c.needWatchedNamespaceList {
		c.initNamespacedPodInformer(watchedNS)
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

func (c *dubboRegistryController) Run() {

	logger.Debugf("starting namespaced informer-factory")
	for _, factory := range c.namespacedInformerFactory {
		go factory.Start(c.ctx.Done())
	}
	logger.Debugf("finish start namespaced informer-factory")
}

func (c *dubboRegistryController) run() {

	defer c.queue.ShutDown()

	for ns, podInformer := range c.namespacedPodInformers {
		if !cache.WaitForCacheSync(c.ctx.Done(), podInformer.Informer().HasSynced) {
			logger.Errorf("wait for cache sync finish @namespace %s fail", ns)
			return
		}
	}

	logger.Infof("init kubernetes registry client success @namespace = %q @Podname = %q", c.namespace, c.name)

	// start work
	go c.work()
	// block wait context cancel
	<-c.ctx.Done()
}

func (c *dubboRegistryController) work() {
	defer logger.Debugf("dubbo registry controller work stopped")
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

// handleWatchedPodEvent
// handle watched pod event
func (c *dubboRegistryController) handleWatchedPodEvent(p *v1.Pod, eventType watch.EventType) {

	logger.Warnf("get @type = %s event from @pod = %s", eventType , p.GetName())

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

			logger.Debugf("prepare to put object (%#v) to kubernetes-watcherSet", o)

			if err := c.watcherSet.Put(o); err != nil {
				logger.Errorf("put (%#v) to cache watcherSet: %v ", o, err)
				return
			}

		}

	}
}

// unmarshalRecord
// unmarshal the kubernetes dubbo annotation value
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
func (c *dubboRegistryController) initCurrentPod() (*v1.Pod, error) {

	currentPod, err := c.kc.CoreV1().Pods(c.namespace).Get(c.name, metav1.GetOptions{})
	if err != nil {
		return nil, perrors.WithMessagef(err, "get current (%s) pod in namespace (%s)", c.name, c.namespace)
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

// patch current pod
// write new meta for current pod
func (c *dubboRegistryController) patchCurrentPod(patch []byte) (*v1.Pod, error) {

	updatedPod, err := c.kc.CoreV1().Pods(c.namespace).Patch(c.name, types.StrategicMergePatchType, patch)
	if err != nil {
		return nil, perrors.WithMessage(err, "patch in kubernetes pod ")
	}
	return updatedPod, nil
}

// assemble the dubbo kubernetes label
// every dubbo instance should be labeled spec {"dubbo.io/label":"dubbo.io/label-value"} label
func (c *dubboRegistryController) assembleDUBBOLabel(currentPod *v1.Pod) (*v1.Pod, *v1.Pod, error) {

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

// getPatch
// get the kubernetes pod patch bytes
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

// marshalRecord
// marshal the kubernetes dubbo annotation value
func (c *dubboRegistryController) marshalRecord(ol []*WatcherEvent) (string, error) {

	msg, err := json.Marshal(ol)
	if err != nil {
		return "", perrors.WithMessage(err, "json encode object list")
	}
	return base64.URLEncoding.EncodeToString(msg), nil
}

func (c *dubboRegistryController) readCurrentPod() (*v1.Pod, error) {

	currentPod, err := c.kc.CoreV1().Pods(c.namespace).Get(c.name, metav1.GetOptions{})
	if err != nil {
		return nil, perrors.WithMessagef(err, "get current (%s) pod in namespace (%s)", c.name, c.namespace)
	}
	return currentPod, nil
}

func (c *dubboRegistryController) addAnnotationForCurrentPod(k string, v string) error {

	c.lock.Lock()
	defer c.lock.Unlock()

	// 1. accord old pod && (k, v) assemble new pod dubbo annotion v
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
	return nil
}
