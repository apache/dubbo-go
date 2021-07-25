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

package generalizer

import (
	"reflect"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	protobufJsonGeneralizer     Generalizer
	protobufJsonGeneralizerOnce sync.Once
)

func GetProtobufJsonGeneralizer() Generalizer {
	protobufJsonGeneralizerOnce.Do(func() {
		protobufJsonGeneralizer = &ProtobufJsonGeneralizer{}
	})
	return protobufJsonGeneralizer
}

// ProtobufJsonGeneralizer generalizes an object to json and realizes an object from json using protobuf.
// Currently, ProtobufJsonGeneralizer is disabled temporarily until the triple protocol is ready.
type ProtobufJsonGeneralizer struct{}

func (g *ProtobufJsonGeneralizer) Generalize(obj interface{}) (interface{}, error) {
	message, ok := obj.(proto.Message)
	if !ok {
		return nil, perrors.Errorf("unexpected type of obj(=%T), wanted is proto.Message", obj)
	}

	jsonbytes, err := protojson.Marshal(message)
	if err != nil {
		return nil, err
	}

	return string(jsonbytes), nil
}

func (g *ProtobufJsonGeneralizer) Realize(obj interface{}, typ reflect.Type) (interface{}, error) {
	jsonbytes, ok := obj.(string)
	if !ok {
		return nil, perrors.Errorf("unexpected type of obj(=%T), wanted is string", obj)
	}

	// typ represents a struct instead of a pointer
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// create the target object
	ret, ok := reflect.New(typ).Interface().(proto.Message)
	if !ok {
		return nil, perrors.Errorf("the type of obj(=%s) should be proto.Message", typ)
	}

	// get the values from json
	err := protojson.Unmarshal([]byte(jsonbytes), ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// GetType returns empty string for "protobuf-json"
func (g *ProtobufJsonGeneralizer) GetType(_ interface{}) (string, error) {
	return "", nil
}
