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

// Package base implements invoker for the manipulation of cluster strategy.
package base

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type BaseClusterInvoker struct {
	Directory      directory.Directory
	AvailableCheck bool
	Destroyed      *atomic.Bool
	StickyInvoker  protocol.Invoker
}

func NewBaseClusterInvoker(directory directory.Directory) BaseClusterInvoker {
	return BaseClusterInvoker{
		Directory:      directory,
		AvailableCheck: true,
		Destroyed:      atomic.NewBool(false),
	}
}

func (invoker *BaseClusterInvoker) GetURL() *common.URL {
	return invoker.Directory.GetURL()
}

func (invoker *BaseClusterInvoker) Destroy() {
	// this is must atom operation
	if invoker.Destroyed.CAS(false, true) {
		invoker.Directory.Destroy()
	}
}

func (invoker *BaseClusterInvoker) IsAvailable() bool {
	if invoker.StickyInvoker != nil {
		return invoker.StickyInvoker.IsAvailable()
	}
	return invoker.Directory.IsAvailable()
}

// CheckInvokers checks invokers' status if is available or not
func (invoker *BaseClusterInvoker) CheckInvokers(invokers []protocol.Invoker, invocation protocol.Invocation) error {
	if len(invokers) == 0 {
		ip := common.GetLocalIp()
		return perrors.Errorf("Failed to invoke the method %v. No provider available for the service %v from "+
			"registry %v on the consumer %v using the dubbo version %v .Please check if the providers have been started and registered.",
			invocation.MethodName(), invoker.Directory.GetURL().SubURL.Key(), invoker.Directory.GetURL().String(), ip, constant.Version)
	}
	return nil
}

// CheckWhetherDestroyed checks if cluster invoker was destroyed or not
func (invoker *BaseClusterInvoker) CheckWhetherDestroyed() error {
	if invoker.Destroyed.Load() {
		ip := common.GetLocalIp()
		return perrors.Errorf("Rpc cluster invoker for %v on consumer %v use dubbo version %v is now destroyed! can not invoke any more. ",
			invoker.Directory.GetURL().Service(), ip, constant.Version)
	}
	return nil
}

func (invoker *BaseClusterInvoker) DoSelect(lb loadbalance.LoadBalance, invocation protocol.Invocation, invokers []protocol.Invoker, invoked []protocol.Invoker) protocol.Invoker {
	var selectedInvoker protocol.Invoker
	if len(invokers) <= 0 {
		return selectedInvoker
	}

	url := invokers[0].GetURL()
	sticky := url.GetParamBool(constant.StickyKey, false)
	// Get the service method sticky config if have
	sticky = url.GetMethodParamBool(invocation.MethodName(), constant.StickyKey, sticky)

	if invoker.StickyInvoker != nil && !isInvoked(invoker.StickyInvoker, invokers) {
		invoker.StickyInvoker = nil
	}

	if sticky && invoker.AvailableCheck &&
		invoker.StickyInvoker != nil && invoker.StickyInvoker.IsAvailable() &&
		(invoked == nil || !isInvoked(invoker.StickyInvoker, invoked)) {
		return invoker.StickyInvoker
	}

	selectedInvoker = invoker.doSelectInvoker(lb, invocation, invokers, invoked)
	if sticky {
		invoker.StickyInvoker = selectedInvoker
	}
	return selectedInvoker
}

func (invoker *BaseClusterInvoker) doSelectInvoker(lb loadbalance.LoadBalance, invocation protocol.Invocation, invokers []protocol.Invoker, invoked []protocol.Invoker) protocol.Invoker {
	if len(invokers) == 0 {
		return nil
	}
	go protocol.TryRefreshBlackList()
	if len(invokers) == 1 {
		if invokers[0].IsAvailable() {
			return invokers[0]
		}
		protocol.SetInvokerUnhealthyStatus(invokers[0])
		logger.Errorf("the invokers of %s is nil. ", invokers[0].GetURL().ServiceKey())
		return nil
	}

	selectedInvoker := lb.Select(invokers, invocation)

	// judge if the selected Invoker is invoked and available
	if (!selectedInvoker.IsAvailable() && invoker.AvailableCheck) || isInvoked(selectedInvoker, invoked) {
		protocol.SetInvokerUnhealthyStatus(selectedInvoker)
		otherInvokers := getOtherInvokers(invokers, selectedInvoker)
		// do reselect
		for i := 0; i < 3; i++ {
			if len(otherInvokers) == 0 {
				// no other ivk to reselect, return to fallback
				break
			}
			reselectedInvoker := lb.Select(otherInvokers, invocation)
			if isInvoked(reselectedInvoker, invoked) {
				otherInvokers = getOtherInvokers(otherInvokers, reselectedInvoker)
				continue
			}
			if !reselectedInvoker.IsAvailable() {
				logger.Infof("the invoker of %s is not available, maybe some network error happened or the server is shutdown.",
					invoker.GetURL().Ip)
				protocol.SetInvokerUnhealthyStatus(reselectedInvoker)
				otherInvokers = getOtherInvokers(otherInvokers, reselectedInvoker)
				continue
			}
			return reselectedInvoker
		}
	}

	return selectedInvoker
}

func isInvoked(selectedInvoker protocol.Invoker, invoked []protocol.Invoker) bool {
	for _, i := range invoked {
		if i == selectedInvoker {
			return true
		}
	}
	return false
}

func GetLoadBalance(invoker protocol.Invoker, methodName string) loadbalance.LoadBalance {
	url := invoker.GetURL()

	// Get the service loadbalance config
	lb := url.GetParam(constant.LoadbalanceKey, constant.DefaultLoadBalance)

	// Get the service method loadbalance config if have
	if v := url.GetMethodParam(methodName, constant.LoadbalanceKey, ""); len(v) > 0 {
		lb = v
	}
	return extension.GetLoadbalance(lb)
}

func getOtherInvokers(invokers []protocol.Invoker, invoker protocol.Invoker) []protocol.Invoker {
	otherInvokers := make([]protocol.Invoker, 0)
	for _, i := range invokers {
		if i != invoker {
			otherInvokers = append(otherInvokers, i)
		}
	}
	return otherInvokers
}
