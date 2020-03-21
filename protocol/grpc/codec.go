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
package grpc


import (
	"bytes"
	"encoding/json"
)

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/encoding"
)

func init() {
	encoding.RegisterCodec(JSON{
		Marshaler: jsonpb.Marshaler{
			EmitDefaults: true,
			OrigName:     true,
		},
	})
}

type JSON struct {
	jsonpb.Marshaler
	jsonpb.Unmarshaler
}

func (_ JSON) Name() string {
	return "json"
}

func (j JSON) Marshal(v interface{}) (out []byte, err error) {
	if pm, ok := v.(proto.Message); ok {
		b := new(bytes.Buffer)
		err := j.Marshaler.Marshal(b, pm)
		if err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	}
	return json.Marshal(v)
}

func (j JSON) Unmarshal(data []byte, v interface{}) (err error) {
	if pm, ok := v.(proto.Message); ok {
		b := bytes.NewBuffer(data)
		return j.Unmarshaler.Unmarshal(b, pm)
	}
	return json.Unmarshal(data, v)
}
