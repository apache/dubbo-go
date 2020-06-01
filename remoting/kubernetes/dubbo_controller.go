package kubernetes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
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
	ctx    context.Context
	cancel context.CancelFunc

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

// read dubbo-registry controller config
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

	// init all watch needed pod-informer
	for _, watchedNS := range c.needWatchedNamespaceList {
		c.initNamespacedPodInformer(watchedNS)
	}

	return nil
}

func newDubboRegistryController() error {

	c := &dubboRegistryController{}
	if err := c.readConfig(); err != nil {
		return perrors.WithMessage(err, "dubbo registry controller read config")
	}

	if err := c.init(); err != nil {
		return perrors.WithMessage(err, "dubbo registry controller init")
	}

	go c.run()
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
	for _, factory := range c.namespacedInformerFactory {
		go factory.Start(c.ctx.Done())
	}
}

func (c *dubboRegistryController) stop() {
	c.cancel()
}
func (c *dubboRegistryController) run() {

	defer c.queue.ShutDown()

	for ns, podInformer := range c.namespacedPodInformers {
		if !cache.WaitForCacheSync(c.ctx.Done(), podInformer.Informer().HasSynced) {
			logger.Errorf("wait for cache sync finish @namespace %s fail", ns)
			return
		}
	}

	// start work
	go c.work()
	// block wait
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
