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

// Package generic provides generic invoke filter.
package generic

import (
	"context"
	"sync"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation2 "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

var (
	genericOnce sync.Once
	instance    *genericFilter
)

func init() {
	extension.SetFilter(constant.GenericFilterKey, newGenericFilter)
}

// genericFilter ensures the structs are converted to maps, this filter is for consumer
type genericFilter struct{}

func newGenericFilter() filter.Filter {
	if instance == nil {
		genericOnce.Do(func() {
			instance = &genericFilter{}
		})
	}
	return instance
}

// Invoke turns the parameters to map for generic method
func (f *genericFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if isCallingToGenericService(invoker, invocation) {

		mtdname := invocation.MethodName()
		oldargs := invocation.Arguments()

		types := make([]string, 0, len(oldargs))
		args := make([]hessian.Object, 0, len(oldargs))

		// get generic info from attachments of invocation, the default value is "true"
		generic := invocation.GetAttachmentWithDefaultValue(constant.GenericKey, constant.GenericSerializationDefault)
		// get generalizer according to value in the `generic`
		g := getGeneralizer(generic)

		for _, arg := range oldargs {
			// use the default generalizer(MapGeneralizer)
			typ, err := g.GetType(arg)
			if err != nil {
				logger.Errorf("failed to get type, %v", err)
			}
			obj, err := g.Generalize(arg)
			if err != nil {
				logger.Errorf("generalization failed, %v", err)
				return invoker.Invoke(ctx, invocation)
			}
			types = append(types, typ)
			args = append(args, obj)
		}

		// construct a new invocation for generic call
		newargs := []interface{}{
			mtdname,
			types,
			args,
		}
		newivc := invocation2.NewRPCInvocation(constant.Generic, newargs, invocation.Attachments())
		newivc.SetReply(invocation.Reply())
		newivc.Attachments()[constant.GenericKey] = invoker.GetURL().GetParam(constant.GenericKey, "")

		return invoker.Invoke(ctx, newivc)
	} else if isMakingAGenericCall(invoker, invocation) {
		invocation.Attachments()[constant.GenericKey] = invoker.GetURL().GetParam(constant.GenericKey, "")
	}
	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (f *genericFilter) OnResponse(_ context.Context, result protocol.Result, _ protocol.Invoker,
	_ protocol.Invocation) protocol.Result {
	return result
}
