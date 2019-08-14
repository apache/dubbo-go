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
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/utils"
	"github.com/apache/dubbo-go/protocol"
)

type baseClusterInvoker struct {
	directory      cluster.Directory
	availablecheck bool
	destroyed      *atomic.Bool
}

func newBaseClusterInvoker(directory cluster.Directory) baseClusterInvoker {
	return baseClusterInvoker{
		directory:      directory,
		availablecheck: true,
		destroyed:      atomic.NewBool(false),
	}
}
func (invoker *baseClusterInvoker) GetUrl() common.URL {
	return invoker.directory.GetUrl()
}

func (invoker *baseClusterInvoker) Destroy() {
	//this is must atom operation
	if invoker.destroyed.CAS(false, true) {
		invoker.directory.Destroy()
	}
}

func (invoker *baseClusterInvoker) IsAvailable() bool {
	//TODO:sticky connection
	return invoker.directory.IsAvailable()
}

//check invokers availables
func (invoker *baseClusterInvoker) checkInvokers(invokers []protocol.Invoker, invocation protocol.Invocation) error {
	if len(invokers) == 0 {
		ip, _ := utils.GetLocalIP()
		return perrors.Errorf("Failed to invoke the method %v. No provider available for the service %v from "+
			"registry %v on the consumer %v using the dubbo version %v .Please check if the providers have been started and registered.",
			invocation.MethodName(), invoker.directory.GetUrl().SubURL.Key(), invoker.directory.GetUrl().String(), ip, constant.Version)
	}
	return nil

}

//check cluster invoker is destroyed or not
func (invoker *baseClusterInvoker) checkWhetherDestroyed() error {
	if invoker.destroyed.Load() {
		ip, _ := utils.GetLocalIP()
		return perrors.Errorf("Rpc cluster invoker for %v on consumer %v use dubbo version %v is now destroyed! can not invoke any more. ",
			invoker.directory.GetUrl().Service(), ip, constant.Version)
	}
	return nil
}

func (invoker *baseClusterInvoker) doSelect(lb cluster.LoadBalance, invocation protocol.Invocation, invokers []protocol.Invoker, invoked []protocol.Invoker) protocol.Invoker {
	//todo:sticky connect
	if len(invokers) == 1 {
		return invokers[0]
	}
	selectedInvoker := lb.Select(invokers, invocation)

	//judge to if the selectedInvoker is invoked

	if !selectedInvoker.IsAvailable() || !invoker.availablecheck || isInvoked(selectedInvoker, invoked) {
		// do reselect
		var reslectInvokers []protocol.Invoker

		for _, invoker := range invokers {
			if !invoker.IsAvailable() {
				continue
			}

			if !isInvoked(invoker, invoked) {
				reslectInvokers = append(reslectInvokers, invoker)
			}
		}

		if len(reslectInvokers) > 0 {
			return lb.Select(reslectInvokers, invocation)
		} else {
			return nil
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

func getLoadBalance(invoker protocol.Invoker, invocation protocol.Invocation) cluster.LoadBalance {
	url := invoker.GetUrl()

	methodName := invocation.MethodName()
	//Get the service loadbalance config
	lb := url.GetParam(constant.LOADBALANCE_KEY, constant.DEFAULT_LOADBALANCE)

	//Get the service method loadbalance config if have
	if v := url.GetMethodParam(methodName, constant.LOADBALANCE_KEY, ""); len(v) > 0 {
		lb = v
	}
	return extension.GetLoadbalance(lb)
}
