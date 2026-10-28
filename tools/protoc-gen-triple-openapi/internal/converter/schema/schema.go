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

package schema

import (
	"sort"
)

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func GenerateFileSchemas(tt protoreflect.FileDescriptor) *orderedmap.Map[string, *base.SchemaProxy] {
	messages := make(map[protoreflect.MessageDescriptor]struct{})
	enums := make(map[protoreflect.EnumDescriptor]struct{})
	collectMessages(tt, messages, enums)

	schemas := orderedmap.New[string, *base.SchemaProxy]()

	// It is necessary to sort multiple messages here,
	// otherwise the components will be out of order.
	sortedMessages := make([]protoreflect.MessageDescriptor, 0, len(messages))
	for message := range messages {
		sortedMessages = append(sortedMessages, message)
	}
	sort.Slice(sortedMessages, func(i, j int) bool {
		return sortedMessages[i].FullName() < sortedMessages[j].FullName()
	})

	for _, message := range sortedMessages {
		id, schema := messageToSchema(message)
		if schema != nil {
			schemas.Set(id, base.CreateSchemaProxy(schema))
		}
	}

	sortedEnums := make([]protoreflect.EnumDescriptor, 0, len(enums))
	for enum := range enums {
		sortedEnums = append(sortedEnums, enum)
	}
	sort.Slice(sortedEnums, func(i, j int) bool {
		return sortedEnums[i].FullName() < sortedEnums[j].FullName()
	})

	for _, enum := range sortedEnums {
		id, schema := enumToSchema(enum)
		schemas.Set(id, base.CreateSchemaProxy(schema))
	}

	return schemas
}

func collectMessages(fd protoreflect.FileDescriptor, messages map[protoreflect.MessageDescriptor]struct{}, enums map[protoreflect.EnumDescriptor]struct{}) {
	services := fd.Services()
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		methods := service.Methods()
		for j := 0; j < methods.Len(); j++ {
			method := methods.Get(j)
			collectMethodMessages(method, messages, enums)
		}
	}
}

func collectMethodMessages(md protoreflect.MethodDescriptor, messages map[protoreflect.MessageDescriptor]struct{}, enums map[protoreflect.EnumDescriptor]struct{}) {
	collectMessage(md.Input(), messages, enums)
	collectMessage(md.Output(), messages, enums)
}

func collectMessage(md protoreflect.MessageDescriptor, messages map[protoreflect.MessageDescriptor]struct{}, enums map[protoreflect.EnumDescriptor]struct{}) {
	if md == nil {
		return
	}

	if _, ok := messages[md]; ok {
		return
	}
	messages[md] = struct{}{}

	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		collectField(fields.Get(i), messages, enums)
	}

	nestedMessages := md.Messages()
	for i := 0; i < nestedMessages.Len(); i++ {
		collectMessage(nestedMessages.Get(i), messages, enums)
	}
}

func collectField(fd protoreflect.FieldDescriptor, messages map[protoreflect.MessageDescriptor]struct{}, enums map[protoreflect.EnumDescriptor]struct{}) {
	if fd == nil {
		return
	}
	if fd.Kind() == protoreflect.EnumKind {
		enums[fd.Enum()] = struct{}{}
		return
	}
	collectMessage(fd.Message(), messages, enums)
}
