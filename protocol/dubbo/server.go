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

package dubbo

import (
	"context"
	"fmt"
	"net/url"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/dubbo/impl"
	"github.com/apache/dubbo-go/protocol/dubbo/impl/remoting"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/dubbogo/getty"
	"github.com/opentracing/opentracing-go"
	"gopkg.in/yaml.v2"
)

func init() {

	// load clientconfig from provider_config
	// default use dubbo
	providerConfig := config.GetProviderConfig()
	if providerConfig.ApplicationConfig == nil {
		return
	}
	protocolConf := providerConfig.ProtocolConf
	defaultServerConfig := remoting.GetDefaultServerConfig()
	if protocolConf == nil {
		logger.Info("protocol_conf default use dubbo config")
	} else {
		dubboConf := protocolConf.(map[interface{}]interface{})[DUBBO]
		if dubboConf == nil {
			logger.Warnf("dubboConf is nil")
			return
		}

		dubboConfByte, err := yaml.Marshal(dubboConf)
		if err != nil {
			panic(err)
		}
		err = yaml.Unmarshal(dubboConfByte, &defaultServerConfig)
		if err != nil {
			panic(err)
		}
	}
	remoting.SetServerConfig(defaultServerConfig)
}

// rebuildCtx rebuild the context by attachment.
// Once we decided to transfer more context's key-value, we should change this.
// now we only support rebuild the tracing context
func rebuildCtx(inv *invocation.RPCInvocation) context.Context {
	ctx := context.Background()

	// actually, if user do not use any opentracing framework, the err will not be nil.
	spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.TextMap,
		opentracing.TextMapCarrier(inv.Attachments()))
	if err == nil {
		ctx = context.WithValue(ctx, constant.TRACING_REMOTE_SPAN_CTX, spanCtx)
	}
	return ctx
}

func NewStubHandler() remoting.StubHandler {
	return remoting.StubFunc(func(session getty.Session, p *impl.DubboPackage) {
		u := common.NewURLWithOptions(common.WithPath(p.GetService().Path), common.WithParams(url.Values{}),
			common.WithParamsValue(constant.GROUP_KEY, p.GetService().Group),
			common.WithParamsValue(constant.INTERFACE_KEY, p.GetService().Interface),
			common.WithParamsValue(constant.VERSION_KEY, p.GetService().Version))

		exporter, _ := dubboProtocol.ExporterMap().Load(u.ServiceKey())
		if exporter == nil {
			err := fmt.Errorf("don't have this exporter, key: %s", u.ServiceKey())
			logger.Errorf(err.Error())
			p.SetResponseStatus(impl.Response_OK)
			p.SetBody(err)
			return
		}
		invoker := exporter.(protocol.Exporter).GetInvoker()
		if invoker != nil {
			attachments := p.GetBody().(map[string]interface{})["attachments"].(map[string]string)
			attachments[constant.LOCAL_ADDR] = session.LocalAddr()
			attachments[constant.REMOTE_ADDR] = session.RemoteAddr()

			args := p.GetBody().(map[string]interface{})["args"].([]interface{})
			inv := invocation.NewRPCInvocation(p.GetService().Method, args, attachments)

			ctx := rebuildCtx(inv)
			result := invoker.Invoke(ctx, inv)
			logger.Debugf("invoker result: %+v", result)
			if err := result.Error(); err != nil {
				p.SetResponseStatus(impl.Response_OK)
				p.SetBody(&impl.ResponsePayload{nil, err, result.Attachments()})
			} else {
				res := result.Result()
				p.SetResponseStatus(impl.Response_OK)
				p.SetBody(&impl.ResponsePayload{res, nil, result.Attachments()})
				//logger.Debugf("service return response %v", res)
			}
		}
	})
}
