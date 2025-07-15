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

type collecter struct {
	Messages map[protoreflect.MessageDescriptor]struct{}
}

func NewCollecter() *collecter {
	return &collecter{
		Messages: map[protoreflect.MessageDescriptor]struct{}{},
	}
}

func GenerateFileSchemas(tt protoreflect.FileDescriptor) *orderedmap.Map[string, *base.SchemaProxy] {
	collecter := NewCollecter()

	services := tt.Services()
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		methods := service.Methods()
		for j := 0; j < methods.Len(); j++ {
			method := methods.Get(j)
			collecter.collectMethod(method)
		}
	}

	schemas := orderedmap.New[string, *base.SchemaProxy]()

	sortedMessages := make([]protoreflect.MessageDescriptor, 0, len(collecter.Messages))
	for message := range collecter.Messages {
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

	return schemas
}

func (c *collecter) collectMethod(tt protoreflect.MethodDescriptor) {
	c.collectMessage(tt.Input())
	c.collectMessage(tt.Output())
}

func (c *collecter) collectMessage(tt protoreflect.MessageDescriptor) {
	if tt == nil {
		return
	}

	c.Messages[tt] = struct{}{}

	fields := tt.Fields()
	for i := 0; i < fields.Len(); i++ {
		c.collectField(fields.Get(i))
	}

	// TODO: consider enum

	messages := tt.Messages()
	for i := 0; i < messages.Len(); i++ {
		c.collectMessage(messages.Get(i))
	}
}

func (c *collecter) collectField(tt protoreflect.FieldDescriptor) {
	if tt == nil {
		return
	}
	c.collectMessage(tt.Message())
}
