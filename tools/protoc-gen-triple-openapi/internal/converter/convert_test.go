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

package converter

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

var updateGolden = flag.Bool("update", false, "update OpenAPI golden files")

func TestConvertGolden(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		goldenFile string
	}{
		{
			name:       "YAML",
			format:     "yaml",
			goldenFile: "greet.triple.openapi.yaml",
		},
		{
			name:       "JSON",
			format:     "json",
			goldenFile: "greet.triple.openapi.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := convert(greetRequest(tt.format))
			if err != nil {
				t.Fatalf("convert() error = %v", err)
			}
			if len(response.File) != 1 {
				t.Fatalf("generated %d files, want 1", len(response.File))
			}

			got := response.File[0].GetContent()
			goldenPath := examplePath(t, tt.goldenFile)
			if *updateGolden {
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatalf("update golden file: %v", err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden file: %v", err)
			}
			if got != string(want) {
				t.Errorf("generated OpenAPI differs from %s\n--- want\n%s\n--- got\n%s", tt.goldenFile, want, got)
			}
		})
	}
}

func examplePath(t *testing.T, name string) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source file")
	}
	return filepath.Join(filepath.Dir(sourceFile), "..", "..", "example", name)
}

func greetRequest(format string) *pluginpb.CodeGeneratorRequest {
	const fileName = "example/greet.proto"
	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{fileName},
		Parameter:      proto.String("format=" + format),
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String(fileName),
				Package: proto.String("greet"),
				Syntax:  proto.String("proto3"),
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name: proto.String("GreetingType"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{Name: proto.String("GREETING_TYPE_UNSPECIFIED"), Number: proto.Int32(0)},
							{Name: proto.String("GREETING_TYPE_FORMAL"), Number: proto.Int32(1)},
						},
					},
				},
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("GreetRequest"),
						Field: []*descriptorpb.FieldDescriptorProto{
							field("name", "name", 1, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
							field("aliases", "aliases", 2, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
							field("metadata", "metadata", 3, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".greet.GreetRequest.MetadataEntry"),
							field("type", "type", 4, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, descriptorpb.FieldDescriptorProto_TYPE_ENUM, ".greet.GreetingType"),
							field("profile", "profile", 5, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".greet.GreetProfile"),
						},
						NestedType: []*descriptorpb.DescriptorProto{
							{
								Name: proto.String("MetadataEntry"),
								Field: []*descriptorpb.FieldDescriptorProto{
									field("key", "key", 1, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
									field("value", "value", 2, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
								},
								Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
							},
						},
					},
					{
						Name: proto.String("GreetProfile"),
						Field: []*descriptorpb.FieldDescriptorProto{
							field("salutation", "salutation", 1, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
						},
					},
					{
						Name: proto.String("GreetResponse"),
						Field: []*descriptorpb.FieldDescriptorProto{
							field("greeting", "greeting", 1, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
						},
					},
					{
						Name: proto.String("GreetUserRequest"),
						Field: []*descriptorpb.FieldDescriptorProto{
							field("name", "name", 1, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
						},
					},
					{
						Name: proto.String("GreetUserResponse"),
						Field: []*descriptorpb.FieldDescriptorProto{
							field("greeting", "greeting", 1, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, descriptorpb.FieldDescriptorProto_TYPE_STRING, ""),
						},
					},
				},
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: proto.String("GreetService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{Name: proto.String("Greet"), InputType: proto.String(".greet.GreetRequest"), OutputType: proto.String(".greet.GreetResponse")},
							{Name: proto.String("GreetUser"), InputType: proto.String(".greet.GreetUserRequest"), OutputType: proto.String(".greet.GreetUserResponse")},
						},
					},
				},
			},
		},
	}
}

func field(name, jsonName string, number int32, label descriptorpb.FieldDescriptorProto_Label, kind descriptorpb.FieldDescriptorProto_Type, typeName string) *descriptorpb.FieldDescriptorProto {
	fd := &descriptorpb.FieldDescriptorProto{
		Name:     proto.String(name),
		JsonName: proto.String(jsonName),
		Number:   proto.Int32(number),
		Label:    label.Enum(),
		Type:     kind.Enum(),
	}
	if typeName != "" {
		fd.TypeName = proto.String(typeName)
	}
	return fd
}
