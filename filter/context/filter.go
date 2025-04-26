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

package context

import (
	"context"
	"strings"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	once sync.Once
	ctx  *contextFilter
)

var unloadingKeys = make(map[string]struct{})

func init() {
	unloadingKeys[constant.PathKey] = struct{}{}
	unloadingKeys[constant.InterfaceKey] = struct{}{}
	unloadingKeys[constant.GroupKey] = struct{}{}
	unloadingKeys[constant.VersionKey] = struct{}{}
	unloadingKeys[constant.TokenKey] = struct{}{}
	unloadingKeys[constant.TimeoutKey] = struct{}{}

	unloadingKeys[constant.AsyncKey] = struct{}{}
	unloadingKeys[constant.TagKey] = struct{}{}
	unloadingKeys[constant.ForceUseTag] = struct{}{}

	httpHeaders := []string{
		"accept",
		"accept-charset",
		"accept-datetime",
		"accept-encoding",
		"accept-language",
		"access-control-request-headers",
		"access-control-request-method",
		"authorization",
		"cache-control",
		"connection",
		"content-length",
		"content-md5",
		"content-type",
		"cookie",
		"date",
		"dnt",
		"expect",
		"forwarded",
		"from",
		"host",
		"http2-settings",
		"if-match",
		"if-modified-since",
		"if-none-match",
		"if-range",
		"if-unmodified-since",
		"max-forwards",
		"origin",
		"pragma",
		"proxy-authorization",
		"range",
		"referer",
		"sec-fetch-dest",
		"sec-fetch-mode",
		"sec-fetch-site",
		"sec-fetch-user",
		"te",
		"trailer",
		"upgrade",
		"upgrade-insecure-requests",
		"user-agent",
	}
	for _, header := range httpHeaders {
		unloadingKeys[header] = struct{}{}
	}
}

func init() {
	extension.SetFilter(constant.ContextFilterKey, newContextFilter)
}

type contextFilter struct{}

func newContextFilter() filter.Filter {
	if ctx == nil {
		once.Do(func() {
			ctx = &contextFilter{}
		})
	}
	return ctx
}

// Invoke do nothing
func (f *contextFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return invoker.Invoke(ctx, invocation)
}

// OnResponse pass attachments to result
func (f *contextFilter) OnResponse(ctx context.Context, result protocol.Result, _ protocol.Invoker,
	_ protocol.Invocation) protocol.Result {

	attachments := ctx.Value(constant.AttachmentServerKey).(map[string]any)
	filtered := make(map[string]any)
	for key, value := range attachments {
		if _, exists := unloadingKeys[strings.ToLower(key)]; !exists {
			filtered[key] = value
		}
	}

	result.SetAttachments(filtered)

	return result
}
