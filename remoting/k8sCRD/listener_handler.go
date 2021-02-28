package k8sCRD

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type ListenerHandler interface {
	AddFunc(obj interface{})
	UpdateFunc(oldObj interface{}, newObj interface{})
	DeleteFunc(obj interface{})
	Watch(opts v1.ListOptions, restClient *rest.RESTClient, ns string) (watch.Interface, error)
	List(opts v1.ListOptions, restClient *rest.RESTClient, ns string) (runtime.Object, error)
	GetObject() runtime.Object
}
