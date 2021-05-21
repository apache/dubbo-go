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

package cluster_impl

import (
	perrors "github.com/pkg/errors"
	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type baseClusterInvoker struct {
	directory      cluster.Directory
	availablecheck bool
	destroyed      *atomic.Bool
	stickyInvoker  protocol.Invoker
}

func newBaseClusterInvoker(directory cluster.Directory) baseClusterInvoker {
	return baseClusterInvoker{
		directory:      directory,
		availablecheck: true,
		destroyed:      atomic.NewBool(false),
	}
}

func (invoker *baseClusterInvoker) GetURL() *common.URL {
	return invoker.directory.GetURL()
}

func (invoker *baseClusterInvoker) Destroy() {
	// this is must atom operation
	if invoker.destroyed.CAS(false, true) {
		invoker.directory.Destroy()
	}
}

func (invoker *baseClusterInvoker) IsAvailable() bool {
	if invoker.stickyInvoker != nil {
		return invoker.stickyInvoker.IsAvailable()
	}
	return invoker.directory.IsAvailable()
}

// check invokers availables
func (invoker *baseClusterInvoker) checkInvokers(invokers []protocol.Invoker, invocation protocol.Invocation) error {
	if len(invokers) == 0 {
		ip := common.GetLocalIp()
		return perrors.Errorf("Failed to invoke the method %v. No provider available for the service %v from "+
			"registry %v on the consumer %v using the dubbo version %v .Please check if the providers have been started and registered.",
			invocation.MethodName(), invoker.directory.GetURL().SubURL.Key(), invoker.directory.GetURL().String(), ip, constant.Version)
	}
	return nil
}

// check cluster invoker is destroyed or not
func (invoker *baseClusterInvoker) checkWhetherDestroyed() error {
	if invoker.destroyed.Load() {
		ip := common.GetLocalIp()
		return perrors.Errorf("Rpc cluster invoker for %v on consumer %v use dubbo version %v is now destroyed! can not invoke any more. ",
			invoker.directory.GetURL().Service(), ip, constant.Version)
	}
	return nil
}

func (invoker *baseClusterInvoker) doSelect(lb cluster.LoadBalance, invocation protocol.Invocation, invokers []protocol.Invoker, invoked []protocol.Invoker) protocol.Invoker {
	var selectedInvoker protocol.Invoker
	if len(invokers) <= 0 {
		return selectedInvoker
	}

	url := invokers[0].GetURL()
	sticky := url.GetParamBool(constant.STICKY_KEY, false)
	// Get the service method sticky config if have
	sticky = url.GetMethodParamBool(invocation.MethodName(), constant.STICKY_KEY, sticky)

	if invoker.stickyInvoker != nil && !isInvoked(invoker.stickyInvoker, invokers) {
		invoker.stickyInvoker = nil
	}

	if sticky && invoker.availablecheck &&
		invoker.stickyInvoker != nil && invoker.stickyInvoker.IsAvailable() &&
		(invoked == nil || !isInvoked(invoker.stickyInvoker, invoked)) {
		return invoker.stickyInvoker
	}

	selectedInvoker = invoker.doSelectInvoker(lb, invocation, invokers, invoked)
	if sticky {
		invoker.stickyInvoker = selectedInvoker
	}
	return selectedInvoker
}

func (invoker *baseClusterInvoker) doSelectInvoker(lb cluster.LoadBalance, invocation protocol.Invocation, invokers []protocol.Invoker, invoked []protocol.Invoker) protocol.Invoker {
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
	if (!selectedInvoker.IsAvailable() && invoker.availablecheck) || isInvoked(selectedInvoker, invoked) {
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
	} else {
		return selectedInvoker
	}
	logger.Errorf("all %d invokers is unavailable for %s.", len(invokers), selectedInvoker.GetURL().String())
	return nil
}

func isInvoked(selectedInvoker protocol.Invoker, invoked []protocol.Invoker) bool {
	for _, i := range invoked {
		if i == selectedInvoker {
			return true
		}
	}
	return false
}

func getLoadBalance(invoker protocol.Invoker, invocation protocol.Invocation) cluster.LoadBalance {
	url := invoker.GetURL()

	methodName := invocation.MethodName()
	// Get the service loadbalance config
	lb := url.GetParam(constant.LOADBALANCE_KEY, constant.DEFAULT_LOADBALANCE)

	// Get the service method loadbalance config if have
	if v := url.GetMethodParam(methodName, constant.LOADBALANCE_KEY, ""); len(v) > 0 {
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
