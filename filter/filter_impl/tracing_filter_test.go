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
	"testing"
)

import (
	"github.com/opentracing/opentracing-go"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

func TestTracingFilterInvoke(t *testing.T) {
	url, _ := common.NewURL(
		"dubbo://:20000/UserProvider?app.version=0.0.1&application=BDTService&bean.name=UserProvider" +
			"&cluster=failover&environment=dev&group=&interface=com.ikurento.user.UserProvider&loadbalance=random&methods.GetUser." +
			"loadbalance=random&methods.GetUser.retries=1&methods.GetUser.weight=0&module=dubbogo+user-info+server&name=" +
			"BDTService&organization=ikurento.com&owner=ZX&registry.role=3&retries=&" +
			"service.filter=echo%2Ctoken%2Caccesslog&timestamp=1569153406&token=934804bf-b007-4174-94eb-96e3e1d60cc7&version=&warmup=100")
	invoker := protocol.NewBaseInvoker(url)

	attach := make(map[string]interface{}, 10)
	inv := invocation.NewRPCInvocation("MethodName", []interface{}{"OK", "Hello"}, attach)
	ctx := context.Background()
	tf := newTracingFilter()

	// do not has any span
	tf.Invoke(ctx, invoker, inv)

	span, ctx := opentracing.StartSpanFromContext(ctx, "Test-Operation")
	defer span.Finish()

	// has previous span
	tf.Invoke(ctx, invoker, inv)

	// has remote ctx
	ctx = context.WithValue(context.Background(), constant.TRACING_REMOTE_SPAN_CTX, span.Context())
	tf.Invoke(ctx, invoker, inv)
}
