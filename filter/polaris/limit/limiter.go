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
	"errors"
	"fmt"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/pkg/flow/data"
	"github.com/polarismesh/polaris-go/pkg/model"
	v1 "github.com/polarismesh/polaris-go/pkg/model/pb/v1"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	remotingpolaris "dubbo.apache.org/dubbo-go/v3/remoting/polaris"
	"dubbo.apache.org/dubbo-go/v3/remoting/polaris/parser"
)

type polarisTpsLimiter struct {
	limitAPI polaris.LimitAPI
}

func (pl *polarisTpsLimiter) IsAllowable(url *common.URL, invocation protocol.Invocation) bool {
	if err := remotingpolaris.Check(); errors.Is(err, remotingpolaris.ErrorNoOpenPolarisAbility) {
		logger.Debug("[TpsLimiter][Polaris] not open polaris ratelimit ability")
		return true
	}

	var err error

	pl.limitAPI, err = remotingpolaris.GetLimiterAPI()
	if err != nil {
		logger.Error("[TpsLimiter][Polaris] create polaris LimitAPI fail : %+v", err)
		return true
	}

	req := pl.buildQuotaRequest(url, invocation)
	if req == nil {
		return true
	}
	logger.Debugf("[TpsLimiter][Polaris] quota req : %+v", req)

	resp, err := pl.limitAPI.GetQuota(req)
	if err != nil {
		logger.Error("[TpsLimiter][Polaris] ns:%s svc:%s get quota fail : %+v", remotingpolaris.GetNamespace(), url.Service(), err)
		return true
	}

	return resp.Get().Code == model.QuotaResultOk
}

func (pl *polarisTpsLimiter) buildQuotaRequest(url *common.URL, invoaction protocol.Invocation) polaris.QuotaRequest {
	ns := remotingpolaris.GetNamespace()
	applicationMode := false
	for _, item := range config.GetRootConfig().Registries {
		if item.Protocol == constant.PolarisKey {
			applicationMode = item.RegistryType == constant.ServiceKey
		}
	}

	svc := url.Interface()
	method := invoaction.MethodName()
	if applicationMode {
		svc = config.GetApplicationConfig().Name
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
		switch item.GetType() {
		case v1.MatchArgument_HEADER:
			if val, ok := attachement[item.GetKey()]; ok {
				req.AddArgument(model.BuildHeaderArgument(item.GetKey(), fmt.Sprintf("%+v", val)))
			}
		case v1.MatchArgument_QUERY:
			if val := parser.ParseArgumentsByExpression(item.GetKey(), arguments); val != nil {
				req.AddArgument(model.BuildQueryArgument(item.GetKey(), fmt.Sprintf("%+v", val)))
			}
		case v1.MatchArgument_CALLER_IP:
			callerIp := url.GetParam(constant.RemoteAddr, "")
			if len(callerIp) != 0 {
				req.AddArgument(model.BuildCallerIPArgument(callerIp))
			}
		case model.ArgumentTypeCallerService:
		}
	}

	return req
}

func (pl *polarisTpsLimiter) buildArguments(req *model.QuotaRequestImpl) ([]*v1.MatchArgument, bool) {
	engine := pl.limitAPI.SDKContext().GetEngine()

	getRuleReq := &data.CommonRateLimitRequest{
		DstService: model.ServiceKey{
			Namespace: req.GetNamespace(),
			Service:   req.GetService(),
		},
		Trigger: model.NotifyTrigger{
			EnableDstRateLimit: true,
		},
		ControlParam: model.ControlParam{
			Timeout: time.Millisecond * 500,
		},
	}

	if err := engine.SyncGetResources(getRuleReq); err != nil {
		logger.Error("[TpsLimiter][Polaris] ns:%s svc:%s get RateLimit Rule fail : %+v", req.GetNamespace(), req.GetService(), err)
		return nil, false
	}

	svcRule := getRuleReq.RateLimitRule
	if svcRule == nil || svcRule.GetValue() == nil {
		logger.Warnf("[TpsLimiter][Polaris] ns:%s svc:%s get RateLimit Rule is nil", req.GetNamespace(), req.GetService())
		return nil, false
	}

	rules, ok := svcRule.GetValue().(*v1.RateLimit)
	if !ok {
		logger.Error("[TpsLimiter][Polaris] ns:%s svc:%s get RateLimit Rule invalid", req.GetNamespace(), req.GetService())
		return nil, false
	}

	ret := make([]*v1.MatchArgument, 0, 4)
	for i := range rules.GetRules() {
		rule := rules.GetRules()[i]
		if len(rule.GetArguments()) == 0 {
			continue
		}

		ret = append(ret, rule.Arguments...)
	}

	return ret, true
}
