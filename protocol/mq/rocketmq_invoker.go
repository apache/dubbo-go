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

package mq

import (
	"context"
	"reflect"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/dubbogo/grpc-go/metadata"
	tripleConstant "github.com/dubbogo/triple/pkg/common/constant"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// same as dubbo_invoker.go attachmentKey
var attachmentKey = []string{
	constant.InterfaceKey, constant.GroupKey, constant.TokenKey, constant.TimeoutKey,
	constant.VersionKey,
}

// RocketMQInvoker is implement of protocol.Invoker, a RocketMQInvoker refer to one service and ip.
type RocketMQInvoker struct {
	protocol.BaseInvoker

	topic    string
	producer rocketmq.Producer
	quitOnce sync.Once
	mtx      sync.RWMutex
}

// NewRocketMQInvoker constructor
func NewRocketMQInvoker(url *common.URL) (*RocketMQInvoker, error) {
	// todo 从配置文件中获取
	groupName := "please_rename_unique_group_name"
	nameserverIpPort := []string{"127.0.0.1:9876"}
	nameserverResolver := primitive.NewPassthroughResolver(nameserverIpPort)
	retry := 2

	rocketmqProducer, err := rocketmq.NewProducer(
		producer.WithGroupName(groupName),
		producer.WithNsResolver(nameserverResolver),
		producer.WithRetry(retry),
	)

	if err != nil {
		logger.Warnf("init rocketmq producer groupName:%s nameserver:%v error:%v", groupName, nameserverIpPort, err)
		return nil, err
	}

	if err := rocketmqProducer.Start(); err != nil {
		logger.Warnf("start rocketmq producer groupName:%s nameserver:%v error:%v", groupName, nameserverIpPort, err)
		return nil, err
	}

	return &RocketMQInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		producer:    rocketmqProducer,
	}, nil
}

func (invoker *RocketMQInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	var result protocol.RPCResult

	if invoker.BaseInvoker.IsAvailable() {
		logger.Warnf("rocketmq invoker destroyed")
		result.Err = protocol.ErrDestroyedInvoker
		return &result
	}

	invoker.mtx.RLock()
	defer invoker.mtx.RUnlock()
	if invoker.producer == nil {
		result.Err = protocol.ErrClientClosed
		return &result
	}

	if invoker.BaseInvoker.IsAvailable() {
		logger.Warnf("rocketmq invoker destroyed")
		result.Err = protocol.ErrDestroyedInvoker
		return &result
	}

	for _, k := range attachmentKey {
		if v := di.GetURL().GetParam(k, ""); len(v) > 0 {
			invocation.SetAttachment(k, v)
		}
	}

	// append interface id to ctx
	gRPCMD := make(metadata.MD, 0)
	for k, v := range invocation.Attachments() {
		if str, ok := v.(string); ok {
			gRPCMD.Set(k, str)
			continue
		}
		if str, ok := v.([]string); ok {
			gRPCMD.Set(k, str...)
			continue
		}
		logger.Warnf("triple attachment value with key = %s is invalid, which should be string or []string", k)
	}
	ctx = metadata.NewOutgoingContext(ctx, gRPCMD)
	ctx = context.WithValue(ctx, tripleConstant.InterfaceKey, invoker.BaseInvoker.GetURL().GetParam(constant.InterfaceKey, ""))
	in := make([]reflect.Value, 0, 16)
	in = append(in, reflect.ValueOf(ctx))

	if len(invocation.ParameterValues()) > 0 {
		in = append(in, invocation.ParameterValues()...)
	}

	methodName := invocation.MethodName()
	triAttachmentWithErr := invoker.pro(methodName, in, invocation.Reply())
	result.Err = triAttachmentWithErr.GetError()
	result.Attrs = make(map[string]interface{})
	for k, v := range triAttachmentWithErr.GetAttachments() {
		result.Attrs[k] = v
	}
	result.Rest = invocation.Reply()
	return &result

	return nil
}

func (invoker *RocketMQInvoker) IsAvailable() bool {
	return false
}

func (invoker *RocketMQInvoker) Destroy() {
}
