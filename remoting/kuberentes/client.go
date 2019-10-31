package kuberentes

import (
	"context"
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
	ErrNilKubernetesClient = perrors.New("kubernetes raw client is nil") // full describe the ERR
)

type Client struct {

	// kubernetes connection config
	cfg *rest.Config

	// the kubernetes interface
	rawClient kubernetes.Interface

	podName   string
	nameSpace string
	// protect the store && currentPod
	lock  sync.RWMutex
	// k is pod name
	store map[string]*v1.Pod

	currentPod *v1.Pod

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

func newClient(namespace string) (*Client, error) {

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, perrors.WithMessage(err, "get in-cluster config")
	}

	rawClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, perrors.WithMessage(err, "new kubernetes client by in cluster config")
	}

	podName, err := getCurrentPodName()
	if err != nil {
		return nil, perrors.WithMessage(err, "get pod name")
	}

	// read the current pod status
	currentPod, err := rawClient.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return nil, perrors.WithMessagef(err, "get pod (%s) in namespace (%s)", podName, namespace)
	}

	logger.Info("init kubernetes registry success")
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		cfg:        cfg,
		rawClient:  rawClient,
		ctx:        ctx,
		currentPod: currentPod,
		cancel:     cancel,
	}

	return c, nil
}

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
			return

		default:
		}

		event, ok := <-wc.ResultChan()
		if !ok{
			logger.Info("watch result chan be stopped")
			return
		}

		p,  ok := event.Object.(*v1.Pod)
		if !ok{
			continue
		}

		for ak, av := range p.GetAnnotations(){
			// not dubbo interest event
			if ak != DubboAnnotationKey{
				continue
			}
		}
	}
}

func (c *Client) handleKubernetesWatchedEvent(event watch.Event){


}

func (c *Client) maintenanceStatus() error {

	wc, err := c.rawClient.CoreV1().Pods(c.nameSpace).Watch(metav1.ListOptions{})
	if err != nil {
		return perrors.WithMessagef(err, "watch  the namespace (%s) pods", c.nameSpace)
	}

	// add wg, grace close the client
	c.wg.Add(1)
	go c.maintenanceStatusLoop(wc)
	return nil
}
func (c *Client) Create(k, value string) {}
