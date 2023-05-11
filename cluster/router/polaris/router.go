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

package polaris

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/pkg/model"
	v1 "github.com/polarismesh/polaris-go/pkg/model/pb/v1"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/router"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	remotingpolaris "dubbo.apache.org/dubbo-go/v3/remoting/polaris"
	"dubbo.apache.org/dubbo-go/v3/remoting/polaris/parser"
)

var (
	_ router.PriorityRouter = (*polarisRouter)(nil)
)

var (
	ErrorPolarisServiceRouteRuleEmpty = errors.New("service route rule is empty")
)

func newPolarisRouter() (*polarisRouter, error) {
	if err := remotingpolaris.Check(); errors.Is(err, remotingpolaris.ErrorNoOpenPolarisAbility) {
		return &polarisRouter{
			openRoute: false,
		}, nil
	}

	routerAPI, err := remotingpolaris.GetRouterAPI()
	if err != nil {
		return nil, err
	}
	consumerAPI, err := remotingpolaris.GetConsumerAPI()
	if err != nil {
		return nil, err
	}

	return &polarisRouter{
		openRoute:   true,
		routerAPI:   routerAPI,
		consumerAPI: consumerAPI,
	}, nil
}

type polarisRouter struct {
	openRoute bool

	routerAPI   polaris.RouterAPI
	consumerAPI polaris.ConsumerAPI

	cancel context.CancelFunc

	lock      sync.RWMutex
	instances map[string]model.Instance
}

// Route Determine the target invokers list.
func (p *polarisRouter) Route(invokers []protocol.Invoker, url *common.URL,
	invoaction protocol.Invocation) []protocol.Invoker {

	if !p.openRoute {
		logger.Debug("[Router][Polaris] not open polaris route ability")
		return invokers
	}

	if len(invokers) == 0 {
		logger.Warn("[Router][Polaris] invokers from previous router is empty")
		return invokers
	}

	service := getService(url)
	instanceMap := p.buildInstanceMap(service)
	if len(instanceMap) == 0 {
		return invokers
	}

	invokersMap := make(map[string]protocol.Invoker, len(invokers))
	targetIns := make([]model.Instance, 0, len(invokers))
	for i := range invokers {
		invoker := invokers[i]
		instanceID := invoker.GetURL().GetParam(constant.PolarisInstanceID, "")
		if len(instanceID) == 0 {
			continue
		}
		invokersMap[instanceID] = invoker
		if val, ok := instanceMap[instanceID]; ok {
			targetIns = append(targetIns, val)
		}
	}

	req, err := p.buildRouteRequest(service, url, invoaction)
	if err != nil {
		return invokers
	}
	req.DstInstances = model.NewDefaultServiceInstances(model.ServiceInfo{
		Service:   service,
		Namespace: remotingpolaris.GetNamespace(),
	}, targetIns)

	resp, err := p.routerAPI.ProcessRouters(&req)
	if err != nil {
		return invokers
	}

	ret := make([]protocol.Invoker, 0, len(resp.GetInstances()))
	for i := range resp.GetInstances() {
		if val, ok := invokersMap[resp.GetInstances()[i].GetId()]; ok {
			ret = append(ret, val)
		}
	}

	return ret
}

func getService(url *common.URL) string {
	applicationMode := false
	for _, item := range config.GetRootConfig().Registries {
		if item.Protocol == constant.PolarisKey {
			applicationMode = item.RegistryType == constant.ServiceKey
		}
	}

	service := url.Interface()
	if applicationMode {
		service = config.GetApplicationConfig().Name
	}

	return service
}

func (p *polarisRouter) buildRouteRequest(svc string, url *common.URL,
	invocation protocol.Invocation) (polaris.ProcessRoutersRequest, error) {

	routeReq := polaris.ProcessRoutersRequest{
		ProcessRoutersRequest: model.ProcessRoutersRequest{
			SourceService: model.ServiceInfo{
				Metadata: map[string]string{},
			},
		},
	}

	attachement := invocation.Attachments()
	arguments := invocation.Arguments()

	labels, err := p.buildTrafficLabels(svc)
	if err != nil {
		return polaris.ProcessRoutersRequest{}, err
	}

	for i := range labels {
		label := labels[i]
		if strings.Compare(label, model.LabelKeyPath) == 0 {
			routeReq.AddArguments(model.BuildPathArgument(getInvokeMethod(url, invocation)))
			continue
		}
		if strings.HasPrefix(label, model.LabelKeyHeader) {
			if val, ok := attachement[strings.TrimPrefix(label, model.LabelKeyHeader)]; ok {
				routeReq.SourceService.Metadata[label] = fmt.Sprintf("%+v", val)
				routeReq.AddArguments(model.BuildArgumentFromLabel(label, fmt.Sprintf("%+v", val)))
			}
		}
		if strings.HasPrefix(label, model.LabelKeyQuery) {
			if val := parser.ParseArgumentsByExpression(label, arguments); val != nil {
				routeReq.AddArguments(model.BuildArgumentFromLabel(label, fmt.Sprintf("%+v", val)))
			}
		}
	}

	return routeReq, nil
}

func (p *polarisRouter) buildTrafficLabels(svc string) ([]string, error) {
	req := &model.GetServiceRuleRequest{}
	req.Namespace = remotingpolaris.GetNamespace()
	req.Service = svc
	req.SetTimeout(time.Second)
	engine := p.routerAPI.SDKContext().GetEngine()
	resp, err := engine.SyncGetServiceRule(model.EventRouting, req)
	if err != nil {
		logger.Errorf("[Router][Polaris] ns:%s svc:%s get route rule fail : %+v", req.GetNamespace(), req.GetService(), err)
		return nil, err
	}

	if resp == nil || resp.GetValue() == nil {
		logger.Errorf("[Router][Polaris] ns:%s svc:%s get route rule empty", req.GetNamespace(), req.GetService())
		return nil, ErrorPolarisServiceRouteRuleEmpty
	}

	routeRule := resp.GetValue().(*v1.Routing)
	labels := make([]string, 0, 4)
	labels = append(labels, collectRouteLabels(routeRule.GetInbounds())...)
	labels = append(labels, collectRouteLabels(routeRule.GetInbounds())...)

	return labels, nil
}

func getInvokeMethod(url *common.URL, invoaction protocol.Invocation) string {
	applicationMode := false
	for _, item := range config.GetRootConfig().Registries {
		if item.Protocol == constant.PolarisKey {
			applicationMode = item.RegistryType == constant.ServiceKey
		}
	}

	method := invoaction.MethodName()
	if applicationMode {
		method = url.Interface() + "/" + invoaction.MethodName()
	}

	return method
}

func collectRouteLabels(routings []*v1.Route) []string {
	ret := make([]string, 0, 4)

	for i := range routings {
		route := routings[i]
		sources := route.GetSources()
		for p := range sources {
			source := sources[p]
			for k := range source.GetMetadata() {
				ret = append(ret, k)
			}
		}
	}

	return ret
}

func (p *polarisRouter) buildInstanceMap(svc string) map[string]model.Instance {
	resp, err := p.consumerAPI.GetAllInstances(&polaris.GetAllInstancesRequest{
		GetAllInstancesRequest: model.GetAllInstancesRequest{
			Service:   svc,
			Namespace: remotingpolaris.GetNamespace(),
		},
	})
	if err != nil {
		logger.Errorf("[Router][Polaris] ns:%s svc:%s get all instances fail : %+v", remotingpolaris.GetNamespace(), svc, err)
		return nil
	}

	ret := make(map[string]model.Instance, len(resp.GetInstances()))

	for i := range resp.GetInstances() {
		ret[resp.GetInstances()[i].GetId()] = resp.GetInstances()[i]
	}

	return ret
}

// URL Return URL in router
func (p *polarisRouter) URL() *common.URL {
	return nil
}

// Priority Return Priority in router
// 0 to ^int(0) is better
func (p *polarisRouter) Priority() int64 {
	return 0
}

// Notify the router the invoker list
func (p *polarisRouter) Notify(invokers []protocol.Invoker) {
	if !p.openRoute {
		return
	}
	if len(invokers) == 0 {
		return
	}
	service := getService(invokers[0].GetURL())
	if service == "" {
		logger.Error("url service is empty")
		return
	}

	req := &model.GetServiceRuleRequest{}
	req.Namespace = remotingpolaris.GetNamespace()
	req.Service = service
	req.SetTimeout(time.Second)

	engine := p.routerAPI.SDKContext().GetEngine()
	_, err := engine.SyncGetServiceRule(model.EventRouting, req)
	if err != nil {
		logger.Errorf("[Router][Polaris] ns:%s svc:%s get route rule fail : %+v", req.GetNamespace(), req.GetService(), err)
		return
	}

	_, err = p.consumerAPI.GetAllInstances(&polaris.GetAllInstancesRequest{
		GetAllInstancesRequest: model.GetAllInstancesRequest{
			Service:   service,
			Namespace: remotingpolaris.GetNamespace(),
		},
	})
	if err != nil {
		logger.Errorf("[Router][Polaris] ns:%s svc:%s get all instances fail : %+v", req.GetNamespace(), req.GetService(), err)
		return
	}
}
