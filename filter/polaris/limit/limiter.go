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

package limit

import (
	"fmt"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	remotingpolaris "dubbo.apache.org/dubbo-go/v3/remoting/polaris"
	"dubbo.apache.org/dubbo-go/v3/remoting/polaris/parser"
	"github.com/dubbogo/gost/log/logger"
	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/pkg/model"
	v1 "github.com/polarismesh/polaris-go/pkg/model/pb/v1"
)

type polarisTpsLimiter struct {
	limitApi polaris.LimitAPI
}

func (pl *polarisTpsLimiter) IsAllowable(url *common.URL, invocation protocol.Invocation) bool {
	var err error

	pl.limitApi, err = remotingpolaris.GetLimiterAPI()
	if err != nil {
		logger.Error("[TpsLimiter][Polaris] create polaris LimitAPI fail : %+v", err)
		return true
	}

	req := pl.buildQuotaRequest(url, invocation)
	if req != nil {
		return true
	}

	resp, err := pl.limitApi.GetQuota(req)
	if err != nil {
		logger.Error("[TpsLimiter][Polaris] ns:%s svc:%s get quota fail : %+v", remotingpolaris.GetNamespace(), url.Service(), err)
		return true
	}

	return resp.Get().Code == model.QuotaResultOk
}

func (pl *polarisTpsLimiter) buildQuotaRequest(url *common.URL, invoaction protocol.Invocation) polaris.QuotaRequest {
	ns := remotingpolaris.GetNamespace()

	svc := url.Service()
	method := invoaction.MethodName()
	if val := url.GetParam(constant.ApplicationKey, ""); len(val) != 0 {
		svc = val
		method = url.Interface() + "/" + invoaction.MethodName()
	}

	req := polaris.NewQuotaRequest()
	req.SetNamespace(ns)
	req.SetService(svc)
	req.SetMethod(method)

	matchs, ok := pl.buildArguments(req.(*model.QuotaRequestImpl))
	if !ok {
		return nil
	}

	attachement := invoaction.Attachments()
	arguments := invoaction.Arguments()

	for i := range matchs {
		item := matchs[i]
		switch item.ArgumentType() {
		case model.ArgumentTypeHeader:
			if val, ok := attachement[item.Key()]; ok {
				req.AddArgument(model.BuildHeaderArgument(item.Key(), fmt.Sprintf("%+v", val)))
			}
		case model.ArgumentTypeQuery:
			if val := parser.ParseArgumentsByExpression(item.Key(), arguments); val != nil {

			}
		case model.ArgumentTypeCallerIP:
			callerIp := url.GetParam(constant.RemoteAddr, "")
			if len(callerIp) != 0 {
				req.AddArgument(model.BuildCallerIPArgument(callerIp))
			}
		case model.ArgumentTypeCallerService:
		}
	}

	return req
}

func (pl *polarisTpsLimiter) buildArguments(req *model.QuotaRequestImpl) ([]model.Argument, bool) {
	engine := pl.limitApi.SDKContext().GetEngine()

	resp, err := engine.SyncGetServiceRule(model.EventRateLimiting, &model.GetServiceRuleRequest{
		Service:   req.GetService(),
		Namespace: req.GetNamespace(),
	})

	if err != nil {
		logger.Error("[TpsLimiter][Polaris] ns:%s svc:%s get RateLimit Rule fail : %+v", req.GetNamespace(), req.GetService(), err)
		return nil, false
	}

	rules, ok := resp.GetValue().(*v1.RateLimit)
	if !ok {
		logger.Error("[TpsLimiter][Polaris] ns:%s svc:%s get RateLimit Rule invalid", req.GetNamespace(), req.GetService())
		return nil, false
	}

	ret := make([]model.Argument, 0, 4)
	for i := range rules.GetRules() {
		rule := rules.GetRules()[i]

		for p := range rule.GetArguments() {
			arg := rule.GetArguments()[p]

			ret = append(ret, model.BuildArgumentFromLabel(arg.GetKey(), arg.GetValue().GetValue().GetValue()))
		}
	}

	return ret, true
}
