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

package openapi

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/internal/httpbinding"
)

func TestDefinitionResolver_HTTPBindingOperationContract(t *testing.T) {
	registerOpenAPIHTTPDescriptor.Do(func() {
		file, err := protodesc.NewFile(openAPIHTTPFileDescriptor(), protoregistry.GlobalFiles)
		if err != nil {
			t.Fatalf("build HTTP descriptor: %v", err)
		}
		if err := protoregistry.GlobalFiles.RegisterFile(file); err != nil {
			t.Fatalf("register HTTP descriptor: %v", err)
		}
	})

	const interfaceName = "triple.openapi.test.Library"
	const methodName = "UpdateBook"
	bindings, err := httpbinding.Resolve(protoreflect.FullName(interfaceName + "." + methodName))
	require.NoError(t, err)

	resolver := NewDefinitionResolver(global.DefaultOpenAPIConfig(), true)
	document := resolver.Resolve(interfaceName, &serviceInfo{Methods: []serviceMethodInfo{{
		Name:        methodName,
		ReqInitFunc: func() any { return &openAPIHTTPRequest{} },
		Meta:        map[string]any{"response.type": reflect.TypeFor[openAPIHTTPResponse]()},
	}}})

	want := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		key := strings.ToUpper(binding.Method) + " " + normalizeHTTPPath(binding.PathTemplate)
		want = append(want, key)
		pathItem := document.Paths[normalizeHTTPPath(binding.PathTemplate)]
		require.NotNil(t, pathItem, "missing OpenAPI path for %s", key)
		op := pathItem.GetOperation(strings.ToUpper(binding.Method))
		require.NotNil(t, op, "missing OpenAPI operation for %s", key)
		require.Equal(t, bindingOperationID(interfaceName, methodName, binding.Index), op.OperationId)
	}

	got := make([]string, 0)
	for path, pathItem := range document.Paths {
		for method := range pathItem.GetOperations() {
			got = append(got, method+" "+path)
		}
	}
	sort.Strings(want)
	sort.Strings(got)
	require.Equal(t, want, got)
}
