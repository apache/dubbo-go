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

package k8s_crd

import (
	"sync"
	"time"
)

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/logger"
)

type Client struct {
	rawClient           *rest.RESTClient
	groupVersion        schema.GroupVersion
	namespace           string
	listenerHandlerList []ListenerHandler

	once sync.Once
}

func (c *Client) addKnownTypes(scheme *runtime.Scheme) error {
	for _, v := range c.listenerHandlerList {
		scheme.AddKnownTypes(c.groupVersion,
			v.GetObject(),
		)
	}

	metav1.AddToGroupVersion(scheme, c.groupVersion)
	return nil
}

// NewK8sCRDClient create an K8sCRD client, for target  CRD objects:  @objects
// with given @groupname, @groupVersion, @namespace
// list and watchFunction would be called by k8s informer
func NewK8sCRDClient(groupName, groupVersion, namespace string, handlers ...ListenerHandler) (*Client, error) {
	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()
	if err != nil {
		logger.Warn("InClusterConfig failed, can't get uniform router config from k8s")
		return nil, err
	}

	newClient := &Client{
		listenerHandlerList: handlers,
		namespace:           namespace,
		groupVersion:        schema.GroupVersion{Group: groupName, Version: groupVersion},
	}

	// register object
	SchemeBuilder := runtime.NewSchemeBuilder(newClient.addKnownTypes)

	// add to scheme
	if err = SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		logger.Error("AddToScheme failed in k8s CRD process")
		return nil, err
	}

	// init crd config
	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &newClient.groupVersion
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	newRestClient, err := rest.UnversionedRESTClientFor(&crdConfig)
	if err != nil {
		logger.Error("InClusterConfig failed, can't get uniform router config from k8s")
		return nil, err
	}
	newClient.rawClient = newRestClient
	return newClient, nil
}

// func (c *Client) WatchResources() []cache.Store { can only be called once
func (c *Client) WatchResources() []cache.Store {
	stores := make([]cache.Store, 0)
	c.once.Do(
		func() {
			for _, h := range c.listenerHandlerList {
				projectStore, projectController := cache.NewInformer(
					&cache.ListWatch{
						ListFunc: func(lo metav1.ListOptions) (result runtime.Object, err error) {
							return h.List(lo, c.rawClient, c.namespace)
						},
						WatchFunc: func(lo metav1.ListOptions) (watch.Interface, error) {
							return h.Watch(lo, c.rawClient, c.namespace)
						},
					},
					h.GetObject(),
					time.Second*30, //todo can be configured
					cache.ResourceEventHandlerFuncs{
						AddFunc:    h.AddFunc,
						UpdateFunc: h.UpdateFunc,
						DeleteFunc: h.DeleteFunc,
					},
				)

				go projectController.Run(wait.NeverStop)
				stores = append(stores, projectStore)
			}
		})
	return stores
}
