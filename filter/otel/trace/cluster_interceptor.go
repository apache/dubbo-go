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
	"strings"
	"sync"
)

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

import (
	cluster "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

func init() {
	cluster.SetClusterInterceptor(constant.OTELClientTraceKey, func() cluster.Interceptor {
		return &clusterInterceptor{TracerProvider: otel.GetTracerProvider()}
	})
}

type clusterInterceptor struct{ TracerProvider trace.TracerProvider }

func referenceURL(url *common.URL) *common.URL {
	if url != nil && url.SubURL != nil {
		return url.SubURL
	}
	return url
}

func clientTracingEnabled(url *common.URL) bool {
	if url == nil {
		return false
	}
	enabled := false
	for name := range strings.SplitSeq(url.GetParam(constant.ReferenceFilterKey, ""), ",") {
		switch strings.TrimSpace(name) {
		case "-" + constant.OTELClientTraceKey:
			return false
		case constant.OTELClientTraceKey:
			enabled = true
		}
	}
	return enabled
}

func (i *clusterInterceptor) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	url := referenceURL(invoker.GetURL())
	if !clientTracingEnabled(url) {
		if ctx != nil {
			ctx = cluster.WithDiagnostics(ctx, nil, nil)
		}
		return invoker.Invoke(ctx, invocation)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, span := i.TracerProvider.Tracer(constant.TraceScopeName,
		trace.WithInstrumentationVersion(constant.Version)).Start(ctx,
		invocation.ActualMethodName()+" cluster",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(rpcSpanAttributes(url, invocation.MethodName())...),
		trace.WithAttributes(attribute.String("cluster.strategy", url.GetParam(constant.ClusterKey, constant.ClusterKeyFailover))))
	if !span.IsRecording() {
		defer span.End()
		return invoker.Invoke(cluster.WithDiagnostics(ctx, nil, nil), invocation)
	}
	observation := &logicalSpan{span: span}
	ctx = cluster.WithDiagnostics(ctx, invocation, observation)
	stop := context.AfterFunc(ctx, func() { observation.finish(ctx.Err()) })
	defer func() {
		stop()
		// Also release the span if a downstream invoker panics. The panic
		// continues unchanged; no request/panic payload is added to attributes.
		observation.finish(errors.New("cluster invocation did not complete"))
	}()
	// Only the per-endpoint client filter injects attachments, independently
	// for each attempt. The logical span is propagated through context alone.
	res := invoker.Invoke(ctx, invocation)
	stop()
	var err error
	if res == nil {
		err = errors.New("cluster invocation returned no result")
	} else {
		err = res.Error()
	}
	observation.finish(err)
	return res
}

type logicalSpan struct {
	span       trace.Span
	once       sync.Once
	mu         sync.Mutex
	retries    int
	noProvider bool
	ended      bool
}

func (s *logicalSpan) finish(err error) {
	s.once.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.ended = true
		s.span.SetAttributes(attribute.Int("cluster.retry_count", s.retries))
		if err != nil {
			category := errorType(err)
			if s.noProvider && category == "unknown" {
				category = "service_unavailable"
			}
			recordSpanError(s.span, err, category)
		} else {
			s.span.SetStatus(codes.Ok, codes.Ok.String())
		}
		s.span.End()
	})
}

func (s *logicalSpan) NoProvider() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	s.noProvider = true
	s.span.AddEvent("cluster.no_provider", trace.WithAttributes(attribute.Int("candidate_count", 0)))
}

func (s *logicalSpan) Select(attempt int, lb, peer string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	s.span.AddEvent("cluster.select", trace.WithAttributes(attribute.Int("attempt", attempt),
		attribute.String("lb", lb), attribute.String("peer", peer)))
}

func (s *logicalSpan) Retry(attempt int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	s.retries++
	s.span.AddEvent("cluster.retry", trace.WithAttributes(attribute.Int("attempt", attempt),
		attribute.String("error.type", errorType(err))))
}
