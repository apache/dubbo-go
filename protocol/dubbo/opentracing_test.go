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
	"testing"
)

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// TestFilterContext tests the filterContext function
func TestFilterContext(t *testing.T) {
	tests := []struct {
		desc               string
		attachments        map[string]any
		expectedLen        int
		expectedContains   map[string]string
		notExpectedContain []string
	}{
		{
			desc:               "empty attachments",
			attachments:        make(map[string]any),
			expectedLen:        0,
			expectedContains:   map[string]string{},
			notExpectedContain: []string{},
		},
		{
			desc: "only string values",
			attachments: map[string]any{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			expectedLen: 3,
			expectedContains: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			notExpectedContain: []string{},
		},
		{
			desc: "mixed types",
			attachments: map[string]any{
				"stringKey":        "stringValue",
				"intKey":           123,
				"boolKey":          true,
				"anotherStringKey": "anotherValue",
			},
			expectedLen: 2,
			expectedContains: map[string]string{
				"stringKey":        "stringValue",
				"anotherStringKey": "anotherValue",
			},
			notExpectedContain: []string{"intKey", "boolKey"},
		},
		{
			desc: "with nil values",
			attachments: map[string]any{
				"stringKey": "stringValue",
				"nilKey":    nil,
				"intKey":    456,
			},
			expectedLen: 1,
			expectedContains: map[string]string{
				"stringKey": "stringValue",
			},
			notExpectedContain: []string{"nilKey", "intKey"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := filterContext(test.attachments)
			assert.Equal(t, test.expectedLen, len(result))
			for k, v := range test.expectedContains {
				assert.Equal(t, v, result[k])
			}
			for _, k := range test.notExpectedContain {
				assert.NotContains(t, result, k)
			}
		})
	}
}

// TestFillTraceAttachments tests the fillTraceAttachments function
func TestFillTraceAttachments(t *testing.T) {
	tests := []struct {
		desc                string
		initialAttachments  map[string]any
		traceAttachment     map[string]string
		expectedAttachments map[string]any
	}{
		{
			desc:               "filling empty attachments",
			initialAttachments: make(map[string]any),
			traceAttachment: map[string]string{
				"trace.key1": "trace.value1",
				"trace.key2": "trace.value2",
			},
			expectedAttachments: map[string]any{
				"trace.key1": "trace.value1",
				"trace.key2": "trace.value2",
			},
		},
		{
			desc: "filling attachments with existing values",
			initialAttachments: map[string]any{
				"existing": "value",
			},
			traceAttachment: map[string]string{
				"trace.key1": "trace.value1",
			},
			expectedAttachments: map[string]any{
				"existing":   "value",
				"trace.key1": "trace.value1",
			},
		},
		{
			desc: "with empty trace attachments",
			initialAttachments: map[string]any{
				"key": "value",
			},
			traceAttachment: make(map[string]string),
			expectedAttachments: map[string]any{
				"key": "value",
			},
		},
		{
			desc: "overwriting existing keys",
			initialAttachments: map[string]any{
				"key": "oldValue",
			},
			traceAttachment: map[string]string{
				"key": "newValue",
			},
			expectedAttachments: map[string]any{
				"key": "newValue",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			fillTraceAttachments(test.initialAttachments, test.traceAttachment)
			assert.Equal(t, len(test.expectedAttachments), len(test.initialAttachments))
			for k, v := range test.expectedAttachments {
				assert.Equal(t, v, test.initialAttachments[k])
			}
		})
	}
}

// TestInjectTraceCtx tests the injectTraceCtx function
func TestInjectTraceCtx(t *testing.T) {
	opentracing.SetGlobalTracer(mocktracer.New())

	tests := []struct {
		desc        string
		attachments map[string]any
		expectError bool
	}{
		{
			desc:        "with empty attachments",
			attachments: make(map[string]any),
			expectError: false,
		},
		{
			desc: "with existing string attachments",
			attachments: map[string]any{
				constant.VersionKey: "1.0",
				constant.GroupKey:   "testGroup",
			},
			expectError: false,
		},
		{
			desc: "with mixed type attachments",
			attachments: map[string]any{
				"stringKey": "stringValue",
				"intKey":    123,
				"boolKey":   true,
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			inv := invocation.NewRPCInvocation("testMethod", []any{}, test.attachments)
			span := opentracing.StartSpan("test-span")
			err := injectTraceCtx(span, inv)
			span.Finish()

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, inv.Attachments())
			}
		})
	}
}

// TestInjectTraceCtxIntegration tests the complete flow of filtering and filling trace attachments
func TestInjectTraceCtxIntegration(t *testing.T) {
	mockTracer := mocktracer.New()
	opentracing.SetGlobalTracer(mockTracer)

	tests := []struct {
		desc        string
		attachments map[string]any
		methods     []any
	}{
		{
			desc: "with version and group",
			attachments: map[string]any{
				constant.VersionKey: "1.0.0",
				constant.GroupKey:   "myGroup",
				"customKey":         "customValue",
			},
			methods: []any{"arg1", "arg2"},
		},
		{
			desc: "with minimal attachments",
			attachments: map[string]any{
				constant.VersionKey: "1.0",
			},
			methods: []any{},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			inv := invocation.NewRPCInvocation("myMethod", test.methods, test.attachments)

			span := opentracing.StartSpan("test-operation")
			err := injectTraceCtx(span, inv)
			span.Finish()

			assert.NoError(t, err)
			finalAttachments := inv.Attachments()
			assert.NotNil(t, finalAttachments)

			// Verify original attachments are preserved
			for k, v := range test.attachments {
				assert.Equal(t, v, finalAttachments[k])
			}
		})
	}
}
