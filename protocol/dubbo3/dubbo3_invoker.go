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

package dubbo3

import (
	"context"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dubbogo/grpc-go/metadata"

	tripleConstant "github.com/dubbogo/triple/pkg/common/constant"
	triConfig "github.com/dubbogo/triple/pkg/config"
	"github.com/dubbogo/triple/pkg/triple"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// same as dubbo_invoker.go attachmentKey
var attachmentKey = []string{
	constant.InterfaceKey, constant.GroupKey, constant.TokenKey, constant.TimeoutKey,
	constant.VersionKey,
}

// DubboInvoker is implement of protocol.Invoker, a dubboInvoker refer to one service and ip.
type DubboInvoker struct {
	protocol.BaseInvoker
	// the net layer client, it is focus on network communication.
	client *triple.TripleClient
	// quitOnce is used to make sure DubboInvoker is only destroyed once
	quitOnce sync.Once
	// timeout for service(interface) level.
	timeout time.Duration
	// clientGuard is the client lock of dubbo invoker
	clientGuard *sync.RWMutex
}

// NewDubboInvoker constructor
func NewDubboInvoker(url *common.URL) (*DubboInvoker, error) {
	rt := config.GetConsumerConfig().RequestTimeout

	timeout := url.GetParamDuration(constant.TimeoutKey, rt)
	// for triple pb serialization. The bean name from provider is the provider reference key,
	// which can't locate the target consumer stub, so we use interface key..
	interfaceKey := url.GetParam(constant.InterfaceKey, "")
	consumerService := config.GetConsumerServiceByInterfaceName(interfaceKey)

	dubboSerializaerType := url.GetParam(constant.SerializationKey, constant.ProtobufSerialization)
	triCodecType := tripleConstant.CodecType(dubboSerializaerType)
	// new triple client
	opts := []triConfig.OptionFunction{
		triConfig.WithClientTimeout(uint32(timeout.Seconds())),
		triConfig.WithCodecType(triCodecType),
		triConfig.WithLocation(url.Location),
		triConfig.WithHeaderAppVersion(url.GetParam(constant.AppVersionKey, "")),
		triConfig.WithHeaderGroup(url.GetParam(constant.GroupKey, "")),
		triConfig.WithLogger(logger.GetLogger()),
	}
	if maxCall := url.GetParam(constant.MaxCallRecvMsgSize, ""); maxCall != "" {
		if size, err := strconv.Atoi(maxCall); err == nil && size != 0 {
			opts = append(opts, triConfig.WithGRPCMaxCallRecvMessageSize(size))
		}
	}
	if maxCall := url.GetParam(constant.MaxCallSendMsgSize, ""); maxCall != "" {
		if size, err := strconv.Atoi(maxCall); err == nil && size != 0 {
			opts = append(opts, triConfig.WithGRPCMaxCallSendMessageSize(size))
		}
	}

	tracingKey := url.GetParam(constant.TracingConfigKey, "")
	if tracingKey != "" {
		tracingConfig := config.GetTracingConfig(tracingKey)
		if tracingConfig != nil {
			if tracingConfig.Name == "jaeger" {
				if tracingConfig.ServiceName == "" {
					tracingConfig.ServiceName = config.GetApplicationConfig().Name
				}
				opts = append(opts, triConfig.WithJaegerConfig(
					tracingConfig.Address,
					tracingConfig.ServiceName,
					tracingConfig.UseAgent,
				))
			} else {
				logger.Warnf("unsupported tracing name %s, now triple only support jaeger", tracingConfig.Name)
			}
		}
	}

	triOption := triConfig.NewTripleOption(opts...)
	client, err := triple.NewTripleClient(consumerService, triOption)

	if err != nil {
		return nil, err
	}

	return &DubboInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		client:      client,
		timeout:     timeout,
		clientGuard: &sync.RWMutex{},
	}, nil
}

func (di *DubboInvoker) setClient(client *triple.TripleClient) {
	di.clientGuard.Lock()
	defer di.clientGuard.Unlock()

	di.client = client
}

func (di *DubboInvoker) getClient() *triple.TripleClient {
	di.clientGuard.RLock()
	defer di.clientGuard.RUnlock()

	return di.client
}

// Invoke call remoting.
func (di *DubboInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	var (
		result protocol.RPCResult
	)

	if !di.BaseInvoker.IsAvailable() {
		// Generally, the case will not happen, because the invoker has been removed
		// from the invoker list before destroy,so no new request will enter the destroyed invoker
		logger.Warnf("this dubboInvoker is destroyed")
		result.Err = protocol.ErrDestroyedInvoker
		return &result
	}

	di.clientGuard.RLock()
	defer di.clientGuard.RUnlock()

	if di.client == nil {
		result.Err = protocol.ErrClientClosed
		return &result
	}

	if !di.BaseInvoker.IsAvailable() {
		// Generally, the case will not happen, because the invoker has been removed
		// from the invoker list before destroy,so no new request will enter the destroyed invoker
		logger.Warnf("this grpcInvoker is destroying")
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
	ctx = context.WithValue(ctx, tripleConstant.InterfaceKey, di.BaseInvoker.GetURL().GetParam(constant.InterfaceKey, ""))
	in := make([]reflect.Value, 0, 16)
	in = append(in, reflect.ValueOf(ctx))

	if len(invocation.ParameterValues()) > 0 {
		in = append(in, invocation.ParameterValues()...)
	}

	methodName := invocation.MethodName()
	triAttachmentWithErr := di.client.Invoke(methodName, in, invocation.Reply())
	result.Err = triAttachmentWithErr.GetError()
	result.Attrs = make(map[string]interface{})
	for k, v := range triAttachmentWithErr.GetAttachments() {
		result.Attrs[k] = v
	}
	result.Rest = invocation.Reply()
	return &result
}

// get timeout including methodConfig
func (di *DubboInvoker) getTimeout(invocation *invocation_impl.RPCInvocation) time.Duration {
	timeout := di.GetURL().GetParam(strings.Join([]string{constant.MethodKeys, invocation.MethodName(), constant.TimeoutKey}, "."), "")
	if len(timeout) != 0 {
		if t, err := time.ParseDuration(timeout); err == nil {
			// config timeout into attachment
			invocation.SetAttachment(constant.TimeoutKey, strconv.Itoa(int(t.Milliseconds())))
			return t
		}
	}
	// set timeout into invocation at method level
	invocation.SetAttachment(constant.TimeoutKey, strconv.Itoa(int(di.timeout.Milliseconds())))
	return di.timeout
}

// IsAvailable check if invoker is available, now it is useless
func (di *DubboInvoker) IsAvailable() bool {
	client := di.getClient()
	if client != nil {
		// FIXME here can't check if tcp server is started now!!!
		return client.IsAvailable()
	}
	return false
}

// Destroy destroy dubbo3 client invoker.
func (di *DubboInvoker) Destroy() {
	di.quitOnce.Do(func() {
		di.BaseInvoker.Destroy()
		client := di.getClient()
		if client != nil {
			di.setClient(nil)
			client.Close()
		}
	})
}
