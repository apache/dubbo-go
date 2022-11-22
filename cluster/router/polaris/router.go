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

// TODO need to remove when polaris-go upgrade to v1.3.0
const (
	LabelKeyMethod        = "$method"
	LabelKeyHeader        = "$header."
	LabelKeyQuery         = "$query."
	LabelKeyCallerService = "$caller_service."
	LabelKeyCallerIp      = "$caller_ip"
	LabelKeyPath          = "$path"
	LabelKeyCookie        = "$cookie."
)

var (
	_ router.PriorityRouter = (*polarisRouter)(nil)
)

var (
	ErrorPolarisServiceRouteRuleEmpty = errors.New("service route rule is empty")
)

func newPolarisRouter() (*polarisRouter, error) {
	routerApi, err := remotingpolaris.GetRouterAPI()
	if err != nil {
		return nil, err
	}
	consumerApi, err := remotingpolaris.GetConsumerAPI()
	if err != nil {
		return nil, err
	}

	return &polarisRouter{
		routerApi:   routerApi,
		consumerApi: consumerApi,
	}, nil
}

type polarisRouter struct {
	routerApi   polaris.RouterAPI
	consumerApi polaris.ConsumerAPI

	cancel context.CancelFunc

	lock      sync.RWMutex
	instances map[string]model.Instance
}

// Route Determine the target invokers list.
func (p *polarisRouter) Route(invokers []protocol.Invoker, url *common.URL,
	invoaction protocol.Invocation) []protocol.Invoker {

	if len(invokers) == 0 {
		logger.Warnf("[tag router] invokers from previous router is empty")
		return invokers
	}
	service := url.Service()
	instanceMap := p.buildInstanceMap(service)
	if len(instanceMap) == 0 {
		return invokers
	}

	invokersMap := make(map[string]protocol.Invoker, len(invokers))
	targetIns := make([]model.Instance, 0, len(invokers))
	for i := range invokers {
		invoker := invokers[i]
		instanceId := invoker.GetURL().GetParam(constant.PolarisInstanceID, "")
		if len(instanceId) == 0 {
			continue
		}
		invokersMap[instanceId] = invoker
		if val, ok := instanceMap[instanceId]; ok {
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

	resp, err := p.routerApi.ProcessRouters(&req)
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

func (p *polarisRouter) buildRouteRequest(svc string, url *common.URL,
	invoaction protocol.Invocation) (polaris.ProcessRoutersRequest, error) {

	routeReq := polaris.ProcessRoutersRequest{
		model.ProcessRoutersRequest{
			SourceService: model.ServiceInfo{
				Metadata: map[string]string{},
			},
		},
	}

	attachement := invoaction.Attachments()
	arguments := invoaction.Arguments()

	labels, err := p.buildTrafficLabels(svc)
	if err != nil {
		return polaris.ProcessRoutersRequest{}, err
	}

	for i := range labels {
		label := labels[i]
		if strings.Compare(label, LabelKeyPath) == 0 {
			routeReq.SourceService.Metadata[LabelKeyPath] = getInvokeMethod(url, invoaction)
			continue
		}
		if strings.HasPrefix(label, LabelKeyHeader) {
			if val, ok := attachement[strings.TrimPrefix(label, LabelKeyHeader)]; ok {
				routeReq.SourceService.Metadata[label] = fmt.Sprintf("%+v", val)
			}
		}
		if strings.HasPrefix(label, LabelKeyQuery) {
			if val := parser.ParseArgumentsByExpression(label, arguments); val != nil {
				routeReq.SourceService.Metadata[label] = fmt.Sprintf("%+v", val)
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
	engine := p.routerApi.SDKContext().GetEngine()
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
	resp, err := p.consumerApi.GetAllInstances(&polaris.GetAllInstancesRequest{
		model.GetAllInstancesRequest{
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
	if len(invokers) == 0 {
		return
	}
	service := invokers[0].GetURL().Service()
	if service == "" {
		logger.Error("url service is empty")
		return
	}

	req := &model.GetServiceRuleRequest{}
	req.Namespace = remotingpolaris.GetNamespace()
	req.Service = service
	req.SetTimeout(time.Second)

	engine := p.routerApi.SDKContext().GetEngine()
	_, err := engine.SyncGetServiceRule(model.EventRouting, req)
	if err != nil {
		logger.Errorf("[Router][Polaris] ns:%s svc:%s get route rule fail : %+v", req.GetNamespace(), req.GetService(), err)
		return
	}

	_, err = p.consumerApi.GetAllInstances(&polaris.GetAllInstancesRequest{
		model.GetAllInstancesRequest{
			Service:   service,
			Namespace: remotingpolaris.GetNamespace(),
		},
	})
	if err != nil {
		logger.Errorf("[Router][Polaris] ns:%s svc:%s get all instances fail : %+v", req.GetNamespace(), req.GetService(), err)
		return
	}
}
