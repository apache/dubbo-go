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

package k8s_api

import (
	"fmt"
)

import (
	metav "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

import (
	dubboConfig "github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/remoting"
	"github.com/apache/dubbo-go/remoting/k8sCRD"
)

const VirtualServiceEventKey = "virtualServiceEventKey"
const DestinationRuleEventKey = "destinationRuleEventKe3y"

const VirtualServiceResource = "virtualservices"
const DestRuleResource = "destinationrules"

type VirtualServiceListenerHandler struct {
	listener config_center.ConfigurationListener
}

func (r *VirtualServiceListenerHandler) AddFunc(obj interface{}) {
	fmt.Println("addFunc")
	if vsc, ok := obj.(*dubboConfig.VirtualServiceConfig); ok {
		fmt.Printf("in add func: get asserted VirtualServiceConfig = %+v\n", *vsc)
		event := &config_center.ConfigChangeEvent{
			Key:        VirtualServiceEventKey,
			Value:      vsc,
			ConfigType: remoting.EventTypeAdd,
		}
		r.listener.Process(event)
	}
}

func (r *VirtualServiceListenerHandler) UpdateFunc(oldObj, newObj interface{}) {
	fmt.Println("update func = ")
	if vsc, ok := newObj.(*dubboConfig.VirtualServiceConfig); ok {
		event := &config_center.ConfigChangeEvent{
			Key:        VirtualServiceEventKey,
			Value:      vsc,
			ConfigType: remoting.EventTypeUpdate,
		}
		r.listener.Process(event)
	}

}

func (r *VirtualServiceListenerHandler) DeleteFunc(obj interface{}) {
	if vsc, ok := obj.(*dubboConfig.VirtualServiceConfig); ok {
		event := &config_center.ConfigChangeEvent{
			Key:        VirtualServiceEventKey,
			Value:      vsc,
			ConfigType: remoting.EventTypeDel,
		}
		r.listener.Process(event)
	}

}

func (r *VirtualServiceListenerHandler) Watch(opts metav.ListOptions, restClient *rest.RESTClient, ns string) (watch.Interface, error) {
	fmt.Println("Call Watch")
	opts.Watch = true
	return restClient.
		Get().
		Namespace(ns).
		Resource(VirtualServiceResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

func (r *VirtualServiceListenerHandler) List(opts metav.ListOptions, restClient *rest.RESTClient, ns string) (runtime.Object, error) {
	fmt.Println("Call List")
	result := dubboConfig.VirtualServiceConfigList{}
	err := restClient.
		Get().
		Namespace(ns).
		Resource(VirtualServiceResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

func (r *VirtualServiceListenerHandler) GetObject() runtime.Object {
	return &dubboConfig.VirtualServiceConfig{}
}

func newVirtualServiceListenerHandler(listener config_center.ConfigurationListener) k8sCRD.ListenerHandler {
	return &VirtualServiceListenerHandler{
		listener: listener,
	}
}

type DestRuleListenerHandler struct {
	listener config_center.ConfigurationListener
}

func (r *DestRuleListenerHandler) AddFunc(obj interface{}) {
	fmt.Println("dest rule addFunc")
	if drc, ok := obj.(*dubboConfig.DestinationRuleConfig); ok {
		event := &config_center.ConfigChangeEvent{
			Key:        DestinationRuleEventKey,
			Value:      drc,
			ConfigType: remoting.EventTypeAdd,
		}
		r.listener.Process(event)
	}

}

func (r *DestRuleListenerHandler) UpdateFunc(oldObj, newObj interface{}) {
	fmt.Println("dest rule update func = ")
	if drc, ok := newObj.(*dubboConfig.DestinationRuleConfig); ok {
		event := &config_center.ConfigChangeEvent{
			Key:        DestinationRuleEventKey,
			Value:      drc,
			ConfigType: remoting.EventTypeUpdate,
		}
		r.listener.Process(event)
	}
}

func (r *DestRuleListenerHandler) DeleteFunc(obj interface{}) {
	if drc, ok := obj.(*dubboConfig.DestinationRuleConfig); ok {
		event := &config_center.ConfigChangeEvent{
			Key:        DestinationRuleEventKey,
			Value:      drc,
			ConfigType: remoting.EventTypeDel,
		}
		r.listener.Process(event)
	}
}

func (r *DestRuleListenerHandler) Watch(opts metav.ListOptions, restClient *rest.RESTClient, ns string) (watch.Interface, error) {
	fmt.Println("dest rule Call Watch")
	opts.Watch = true
	return restClient.
		Get().
		Namespace(ns).
		Resource(DestRuleResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

func (r *DestRuleListenerHandler) List(opts metav.ListOptions, restClient *rest.RESTClient, ns string) (runtime.Object, error) {
	fmt.Println("Call List")
	result := dubboConfig.DestinationRuleConfigList{}
	err := restClient.
		Get().
		Namespace(ns).
		Resource(DestRuleResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

func (r *DestRuleListenerHandler) GetObject() runtime.Object {
	return &dubboConfig.DestinationRuleConfig{}
}

func newDestRuleListenerHandler(listener config_center.ConfigurationListener) k8sCRD.ListenerHandler {
	return &DestRuleListenerHandler{
		listener: listener,
	}
}
