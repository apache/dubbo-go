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

// Package hessian2 adapts the existing Hessian library to the public stream SPI.
package hessian2

import (
	"fmt"
	"reflect"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

import (
	hessianresponse "dubbo.apache.org/dubbo-go/v3/protocol/dubbo/hessian2"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

type Codec struct{}

func (*Codec) Name() string { return "hessian2" }
func (c *Codec) Marshal(v any) ([]byte, error) {
	e, _ := c.NewEncoder()
	defer e.Close()
	if err := e.Encode(v); err != nil {
		return nil, err
	}
	return e.Finish()
}
func (c *Codec) Unmarshal(b []byte, v any) error {
	d, _ := c.NewDecoder(b)
	defer d.Close()
	return d.Decode(v)
}
func (*Codec) NewEncoder() (serialization.Encoder, error) {
	return &encoder{inner: hessian.NewEncoder()}, nil
}
func (*Codec) NewDecoder(b []byte) (serialization.Decoder, error) {
	return &decoder{inner: hessian.NewDecoder(b)}, nil
}

type encoder struct{ inner *hessian.Encoder }

func (e *encoder) Encode(v any) error {
	if e.inner == nil {
		return fmt.Errorf("hessian2: closed encoder")
	}
	return e.inner.Encode(v)
}
func (e *encoder) Finish() ([]byte, error) {
	if e.inner == nil {
		return nil, fmt.Errorf("hessian2: closed encoder")
	}
	return append([]byte(nil), e.inner.Buffer()...), nil
}
func (e *encoder) Close() error { e.inner = nil; return nil }

type decoder struct{ inner *hessian.Decoder }

func (d *decoder) Decode(target any) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("hessian2: incompatible target %T: %v", target, p)
		}
	}()
	if d.inner == nil {
		return fmt.Errorf("hessian2: closed decoder")
	}
	rv := reflect.ValueOf(target)
	if !rv.IsValid() || rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("hessian2: target must be a non-nil pointer")
	}
	value, err := d.inner.Decode()
	if err != nil {
		return err
	}
	if value == nil {
		rv.Elem().SetZero()
		return nil
	}
	if target, ok := target.(*map[string]any); ok {
		switch values := value.(type) {
		case map[any]any:
			converted := make(map[string]any, len(values))
			for key, value := range values {
				name, ok := key.(string)
				if !ok {
					return fmt.Errorf("hessian2: attachment key %T is not a string", key)
				}
				converted[name] = value
			}
			*target = converted
			return nil
		case string:
			// Historical Hessian request attachments may be encoded as an empty string.
			if values == "" {
				*target = nil
				return nil
			}
		}
	}
	v := reflect.ValueOf(value)
	if v.Type().AssignableTo(rv.Elem().Type()) {
		rv.Elem().Set(v)
		return nil
	}
	return hessianresponse.ReflectResponse(value, target)
}
func (d *decoder) Close() error { d.inner = nil; return nil }
