// Copyright 2016-2019 hxmhlt
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster_impl

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/common/utils"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	"github.com/dubbo/go-for-apache-dubbo/version"
)

type failoverClusterInvoker struct {
	baseClusterInvoker
}

func newFailoverClusterInvoker(directory cluster.Directory) protocol.Invoker {
	return &failoverClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(directory),
	}
}

func (invoker *failoverClusterInvoker) Invoke(invocation protocol.Invocation) protocol.Result {

	invokers := invoker.directory.List(invocation)
	err := invoker.checkInvokers(invokers, invocation)

	if err != nil {
		return &protocol.RPCResult{Err: err}
	}
	url := invokers[0].GetUrl()

	methodName := invocation.MethodName()
	//Get the service loadbalance config
	lb := url.GetParam(constant.LOADBALANCE_KEY, constant.DEFAULT_LOADBALANCE)

	//Get the service method loadbalance config if have
	if v := url.GetMethodParam(methodName, constant.LOADBALANCE_KEY, ""); v != "" {
		lb = v
	}
	loadbalance := extension.GetLoadbalance(lb)

	//get reties
	retries := url.GetParamInt(constant.RETRIES_KEY, constant.DEFAULT_RETRIES)

	//Get the service method loadbalance config if have
	if v := url.GetMethodParamInt(methodName, constant.RETRIES_KEY, 0); v != 0 {
		retries = v
	}
	invoked := []protocol.Invoker{}
	providers := []string{}
	var result protocol.Result
	for i := int64(0); i < retries; i++ {
		//Reselect before retry to avoid a change of candidate `invokers`.
		//NOTE: if `invokers` changed, then `invoked` also lose accuracy.
		if i > 0 {
			err := invoker.checkWhetherDestroyed()
			if err != nil {
				return &protocol.RPCResult{Err: err}
			}
			invokers = invoker.directory.List(invocation)
			err = invoker.checkInvokers(invokers, invocation)
			if err != nil {
				return &protocol.RPCResult{Err: err}
			}
		}
		ivk := invoker.doSelect(loadbalance, invocation, invokers, invoked)
		invoked = append(invoked, ivk)
		//DO INVOKE
		result = ivk.Invoke(invocation)
		if result.Error() != nil {
			providers = append(providers, ivk.GetUrl().Key())
			continue
		} else {
			return result
		}
	}
	ip, _ := utils.GetLocalIP()
	return &protocol.RPCResult{Err: perrors.Errorf("Failed to invoke the method %v in the service %v. Tried %v times of "+
		"the providers %v (%v/%v)from the registry %v on the consumer %v using the dubbo version %v. Last error is %v.",
		methodName, invoker.GetUrl().Service(), retries, providers, len(providers), len(invokers), invoker.directory.GetUrl(), ip, version.Version, result.Error().Error(),
	)}
}
