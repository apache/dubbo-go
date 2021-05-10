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

package filter_impl

import (
	"context"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	SEATA     = constant.DubboCtxKey("seata")
	SEATA_XID = constant.DubboCtxKey("SEATA_XID")
)

func init() {
	extension.SetFilter(string(SEATA), getSeataFilter)
}

// SeataFilter when use seata-golang, use this filter to transfer xid
type SeataFilter struct{}

// When use Seata, transfer xid by attachments
// Invoke Get Xid by attachment key `SEATA_XID`
func (sf *SeataFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("invoking seata filter.")
	xid := invocation.AttachmentsByKey(string(SEATA_XID), "")
	if strings.TrimSpace(xid) != "" {
		logger.Debugf("Method: %v,Xid: %v", invocation.MethodName(), xid)
		return invoker.Invoke(context.WithValue(ctx, SEATA_XID, xid), invocation)
	}
	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (sf *SeataFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

// getSeataFilter create SeataFilter instance
func getSeataFilter() filter.Filter {
	return &SeataFilter{}
}
