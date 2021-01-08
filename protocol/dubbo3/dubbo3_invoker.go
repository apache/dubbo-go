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
	hessian2 "github.com/apache/dubbo-go-hessian2"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
	invocation_impl "github.com/apache/dubbo-go/protocol/invocation"
	dubbo3 "github.com/dubbogo/triple/pkg/triple"
)

// Dubbo3Invoker is implement of protocol.Invoker. A dubboInvoker refer to one service and ip.
type Dubbo3Invoker struct {
	protocol.BaseInvoker
	// the exchange layer, it is focus on network communication.
	client   *dubbo3.TripleClient
	quitOnce sync.Once
	// timeout for service(interface) level.
	timeout time.Duration
	// Used to record the number of requests. -1 represent this DubboInvoker is destroyed
	reqNum int64
}

// NewDubbo3Invoker constructor
func NewDubbo3Invoker(url *common.URL) (*Dubbo3Invoker, error) {
	requestTimeout := config.GetConsumerConfig().RequestTimeout
	requestTimeoutStr := url.GetParam(constant.TIMEOUT_KEY, config.GetConsumerConfig().Request_Timeout)
	if t, err := time.ParseDuration(requestTimeoutStr); err == nil {
		requestTimeout = t
	}

	client, err := dubbo3.NewTripleClient(url)
	if err != nil {
		return nil, err
	}

	return &Dubbo3Invoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		client:      client,
		reqNum:      0,
		timeout:     requestTimeout,
	}, nil
}

// Invoke call remoting.
func (di *Dubbo3Invoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	var (
		result protocol.RPCResult
	)

	in := make([]reflect.Value, 0, 16)
	in = append(in, reflect.ValueOf(ctx))
	if len(invocation.ParameterValues()) > 0 {
		in = append(in, invocation.ParameterValues()...)
	}

	methodName := invocation.MethodName()
	method := di.client.Invoker.MethodByName(methodName)
	res := method.Call(in)

	result.Rest = res[0]
	// check err
	if !res[1].IsNil() {
		result.Err = res[1].Interface().(error)
	} else {
		_ = hessian2.ReflectResponse(res[0], invocation.Reply())
	}

	return &result
}

// get timeout including methodConfig
func (di *Dubbo3Invoker) getTimeout(invocation *invocation_impl.RPCInvocation) time.Duration {
	timeout := di.GetUrl().GetParam(strings.Join([]string{constant.METHOD_KEYS, invocation.MethodName(), constant.TIMEOUT_KEY}, "."), "")
	if len(timeout) != 0 {
		if t, err := time.ParseDuration(timeout); err == nil {
			// config timeout into attachment
			invocation.SetAttachments(constant.TIMEOUT_KEY, strconv.Itoa(int(t.Milliseconds())))
			return t
		}
	}
	// set timeout into invocation at method level
	invocation.SetAttachments(constant.TIMEOUT_KEY, strconv.Itoa(int(di.timeout.Milliseconds())))
	return di.timeout
}

// IsAvailable check if invoker is available, now it is useless
func (di *Dubbo3Invoker) IsAvailable() bool {
	return di.client.IsAvailable()
}

// Destroy destroy dubbo3 client invoker.
func (di *Dubbo3Invoker) Destroy() {
	di.quitOnce.Do(func() {
		for {
			if di.reqNum == 0 {
				di.reqNum = -1
				logger.Infof("dubboInvoker is destroyed,url:{%s}", di.GetUrl().Key())
				di.BaseInvoker.Destroy()
				if di.client != nil {
					di.client.Close()
					di.client = nil
				}
				break
			}
			logger.Warnf("DubboInvoker is to be destroyed, wait {%v} req end,url:{%s}", di.reqNum, di.GetUrl().Key())
			time.Sleep(1 * time.Second)
		}

	})
}
