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

package trace

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
)

import (
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/trace"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

// errorType mirrors metrics/rpc's precedence without depending on metrics or
// guessing local failure origins from error text. Protocol statuses take
// precedence over wrapped context errors and confirmed local origins.
func errorType(err error) string {
	code := tri.CodeOf(err)
	switch code {
	case tri.CodeDeadlineExceeded:
		return "timeout"
	case tri.CodeResourceExhausted:
		return "limit"
	case tri.CodeUnavailable, tri.CodePermissionDenied:
		if code == tri.CodePermissionDenied || rpcerror.Of(err) == rpcerror.Unknown {
			return "service_unavailable"
		}
	case tri.CodeBizError:
		return "business_failed"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "unknown"
	}
	switch rpcerror.Of(err) {
	case rpcerror.Network:
		return "network_failure"
	case rpcerror.Codec:
		return "codec"
	default:
		return "unknown"
	}
}

// Error URLs are exported as origin-only summaries. Dropping the entire path,
// query and fragment also covers registry-specific or encoded credential keys.
var errorURLPattern = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s]+`)

func safeErrorMessage(message string) string {
	return errorURLPattern.ReplaceAllStringFunc(message, func(raw string) string {
		parsed, err := url.Parse(strings.TrimRight(raw, ".,;)}\"'"))
		if err != nil || parsed.Host == "" {
			return "[redacted URL]"
		}
		return parsed.Scheme + "://" + parsed.Host
	})
}

func recordSpanError(span trace.Span, err error, category string) {
	message := safeErrorMessage(err.Error())
	recordedErr := err
	if message != err.Error() {
		// Only the exported copy changes; callers retain the original error chain.
		recordedErr = errors.New(message)
	}
	span.RecordError(recordedErr)
	span.SetAttributes(attribute.String("error.type", category))
	span.SetStatus(codes.Error, message)
}
