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

package getty

import (
	"github.com/opentracing/opentracing-go"
)

import (
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func injectTraceCtx(currentSpan opentracing.Span, inv *invocation_impl.RPCInvocation) error {
	// inject opentracing ctx
	traceAttachments := filterContext(inv.Attachments())
	carrier := opentracing.TextMapCarrier(traceAttachments)
	err := opentracing.GlobalTracer().Inject(currentSpan.Context(), opentracing.TextMap, carrier)
	if err == nil {
		fillTraceAttachments(inv.Attachments(), traceAttachments)
	}
	return err
}

func filterContext(attachments map[string]interface{}) map[string]string {
	traceAttchment := make(map[string]string)
	for k, v := range attachments {
		if r, ok := v.(string); ok {
			traceAttchment[k] = r
		}
	}
	return traceAttchment
}

func fillTraceAttachments(attachments map[string]interface{}, traceAttachment map[string]string) {
	for k, v := range traceAttachment {
		attachments[k] = v
	}
}
