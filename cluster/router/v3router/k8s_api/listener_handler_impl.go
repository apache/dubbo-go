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
	metav "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router/v3router/k8s_crd"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

const (
	VirtualServiceEventKey  = "virtualServiceEventKey"
	DestinationRuleEventKey = "destinationRuleEventKe3y"

	VirtualServiceResource = "virtualservices"
	DestRuleResource       = "destinationrules"
)

// nolint
type VirtualServiceListenerHandler struct {
	listener config_center.ConfigurationListener
}

// nolint
func (r *VirtualServiceListenerHandler) AddFunc(obj interface{}) {
	if vsc, ok := obj.(*config.VirtualServiceConfig); ok {
		event := &config_center.ConfigChangeEvent{
			Key:        VirtualServiceEventKey,
			Value:      vsc,
			ConfigType: remoting.EventTypeAdd,
		}
		r.listener.Process(event)
	}
}

// nolint
func (r *VirtualServiceListenerHandler) UpdateFunc(oldObj, newObj interface{}) {
	if vsc, ok := newObj.(*config.VirtualServiceConfig); ok {
		event := &config_center.ConfigChangeEvent{
			Key:        VirtualServiceEventKey,
			Value:      vsc,
			ConfigType: remoting.EventTypeUpdate,
		}
		r.listener.Process(event)
	}

}

// nolint
func (r *VirtualServiceListenerHandler) DeleteFunc(obj interface{}) {
	if vsc, ok := obj.(*config.VirtualServiceConfig); ok {
		event := &config_center.ConfigChangeEvent{
			Key:        VirtualServiceEventKey,
			Value:      vsc,
			ConfigType: remoting.EventTypeDel,
		}
		r.listener.Process(event)
	}
}

// nolint
func (r *VirtualServiceListenerHandler) Watch(opts metav.ListOptions, restClient *rest.RESTClient, ns string) (watch.Interface, error) {
	opts.Watch = true
	return restClient.
		Get().
		Namespace(ns).
		Resource(VirtualServiceResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// nolint
func (r *VirtualServiceListenerHandler) List(opts metav.ListOptions, restClient *rest.RESTClient, ns string) (runtime.Object, error) {
	result := config.VirtualServiceConfigList{}
	err := restClient.
		Get().
		Namespace(ns).
		Resource(VirtualServiceResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

// nolint
func (r *VirtualServiceListenerHandler) GetObject() runtime.Object {
	return &config.VirtualServiceConfig{}
}

// nolint
func newVirtualServiceListenerHandler(listener config_center.ConfigurationListener) k8s_crd.ListenerHandler {
	return &VirtualServiceListenerHandler{
		listener: listener,
	}
}

// nolint
type DestRuleListenerHandler struct {
	listener config_center.ConfigurationListener
}

// nolint
func (r *DestRuleListenerHandler) AddFunc(obj interface{}) {
	if drc, ok := obj.(*config.DestinationRuleConfig); ok {
		event := &config_center.ConfigChangeEvent{
			Key:        DestinationRuleEventKey,
			Value:      drc,
			ConfigType: remoting.EventTypeAdd,
		}
		r.listener.Process(event)
	}

}

// nolint
func (r *DestRuleListenerHandler) UpdateFunc(oldObj, newObj interface{}) {
	if drc, ok := newObj.(*config.DestinationRuleConfig); ok {
		event := &config_center.ConfigChangeEvent{
			Key:        DestinationRuleEventKey,
			Value:      drc,
			ConfigType: remoting.EventTypeUpdate,
		}
		r.listener.Process(event)
	}
}

// nolint
func (r *DestRuleListenerHandler) DeleteFunc(obj interface{}) {
	if drc, ok := obj.(*config.DestinationRuleConfig); ok {
		event := &config_center.ConfigChangeEvent{
			Key:        DestinationRuleEventKey,
			Value:      drc,
			ConfigType: remoting.EventTypeDel,
		}
		r.listener.Process(event)
	}
}

// nolint
func (r *DestRuleListenerHandler) Watch(opts metav.ListOptions, restClient *rest.RESTClient, ns string) (watch.Interface, error) {
	opts.Watch = true
	return restClient.
		Get().
		Namespace(ns).
		Resource(DestRuleResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// nolint
func (r *DestRuleListenerHandler) List(opts metav.ListOptions, restClient *rest.RESTClient, ns string) (runtime.Object, error) {
	result := config.DestinationRuleConfigList{}
	err := restClient.
		Get().
		Namespace(ns).
		Resource(DestRuleResource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

// nolint
func (r *DestRuleListenerHandler) GetObject() runtime.Object {
	return &config.DestinationRuleConfig{}
}

func newDestRuleListenerHandler(listener config_center.ConfigurationListener) k8s_crd.ListenerHandler {
	return &DestRuleListenerHandler{
		listener: listener,
	}
}
