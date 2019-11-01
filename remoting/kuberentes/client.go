package kuberentes

import (
	"context"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/etcd"
	"os"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/apache/dubbo-go/common/logger"
	perrors "github.com/pkg/errors"
)

const (
	ConnDelay                = 3
	MaxFailTimes             = 15
	RegistryKubernetesClient = "kubernetes registry"

	// kubernetes inject the var
	podNameKey = "HOSTNAME"
	// all pod annotation key
	DubboAnnotationKey = "DUBBO"
)

var (
	ErrKubernetesClientAlreadyClosed = perrors.New("kubernetes client already be closed")
)

type Client struct {

	// kubernetes connection config
	cfg *rest.Config

	// the kubernetes interface
	rawClient kubernetes.Interface

	// current pod config
	currentPodName   string
	currentPod *v1.Pod

	ns string

	// the memory store
	store Store

	// protect the maintenanceStatus loop
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
		ns: namespace,
		cfg:        cfg,
		rawClient:  rawClient,
		ctx:        ctx,
		store:      newStore(ctx),
		cancel:     cancel,
	}

	// read the current pod status
	currentPods, err := rawClient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, perrors.WithMessagef(err, "list pods  in namespace (%s)",  namespace)
	}

	// init the store by current pods
	c.initStore(currentPods)

	// start kubernetes watch loop
	if err := c.maintenanceStatus(); err != nil{
		return nil, perrors.WithMessage(err, "maintenance the kubernetes status")
	}

	logger.Info("init kubernetes registry success")
	return c, nil
}


// initStore
// init the store
func (c *Client) initStore(pods *v1.PodList){
	for _, pod := range pods.Items{
		c.store[pod.Name] = &pod
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
			logger.Info("the client stopped")
			return
		default:
			// get one element from result-chan
			event, ok := <-wc.ResultChan()
			if !ok {
				logger.Info("watch result chan be stopped")
				return
			}
			go c.handleWatchedEvent(event)
		}
	}
}


func (c *Client) handleWatchedEvent(event watch.Event) {

	p, ok := event.Object.(*v1.Pod)
	if !ok {
		// not a pod info, drop it
		return
	}

	for ak, av := range p.GetAnnotations() {

		// not dubbo interest pod
		if ak != DubboAnnotationKey {
			return
		}

		o :=  Object{
			K: ,
		}
		switch event.Type{
		case watch.Added:
			o.EventType = Create
		case watch.Modified:
			o.EventType = Update
		case watch.Deleted:
			o.EventType = Delete
		case watch.Error:
			logger.Warnf("kubernetes watch api report err (%#v)", event)
			return
		default:
		}
		c.store.Put(&o)
	}
}

func (c *Client) Create(k, value string) {}
