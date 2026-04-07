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

package proxy_factory

import (
	"errors"
	"reflect"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

type callLocalMethodSample struct{}

func (s *callLocalMethodSample) Sum(a, b int) int {
	return a + b
}

func (s *callLocalMethodSample) PanicError() {
	panic(errors.New("boom"))
}

func (s *callLocalMethodSample) PanicString() {
	panic("boom str")
}

func (s *callLocalMethodSample) PanicUnknown() {
	panic(123)
}

func (s *callLocalMethodSample) EchoVariadic(args ...any) []any {
	return args
}

func TestCallLocalMethod(t *testing.T) {
	sample := &callLocalMethodSample{}
	cases := []struct {
		name         string
		method       string
		in           []reflect.Value
		useCallSlice bool
		assertErr    func(t *testing.T, err error)
		assertOut    func(t *testing.T, out []reflect.Value)
	}{
		{
			name:         "call success",
			method:       "Sum",
			in:           []reflect.Value{reflect.ValueOf(sample), reflect.ValueOf(1), reflect.ValueOf(2)},
			useCallSlice: false,
			assertErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
			assertOut: func(t *testing.T, out []reflect.Value) {
				assert.Len(t, out, 1)
				assert.Equal(t, 3, out[0].Interface())
			},
		},
		{
			name:         "panic with error",
			method:       "PanicError",
			in:           []reflect.Value{reflect.ValueOf(sample)},
			useCallSlice: false,
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "boom")
			},
		},
		{
			name:         "panic with string",
			method:       "PanicString",
			in:           []reflect.Value{reflect.ValueOf(sample)},
			useCallSlice: false,
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "boom str")
			},
		},
		{
			name:         "panic with unknown type",
			method:       "PanicUnknown",
			in:           []reflect.Value{reflect.ValueOf(sample)},
			useCallSlice: false,
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "invoke function error, unknow exception: 123")
			},
		},
		{
			name:         "variadic slice stays packed without call slice",
			method:       "EchoVariadic",
			in:           []reflect.Value{reflect.ValueOf(sample), reflect.ValueOf([]any{"alice", "bob"})},
			useCallSlice: false,
			assertErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
			assertOut: func(t *testing.T, out []reflect.Value) {
				assert.Len(t, out, 1)
				assert.Equal(t, []any{[]any{"alice", "bob"}}, out[0].Interface())
			},
		},
		{
			name:         "variadic slice expands with call slice",
			method:       "EchoVariadic",
			in:           []reflect.Value{reflect.ValueOf(sample), reflect.ValueOf([]any{"alice", "bob"})},
			useCallSlice: true,
			assertErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
			assertOut: func(t *testing.T, out []reflect.Value) {
				assert.Len(t, out, 1)
				assert.Equal(t, []any{"alice", "bob"}, out[0].Interface())
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			m, ok := reflect.TypeOf(sample).MethodByName(tt.method)
			if !ok {
				t.Fatalf("method %s not found", tt.method)
			}
			out, err := callLocalMethod(m, tt.in, tt.useCallSlice)
			if tt.assertErr != nil {
				tt.assertErr(t, err)
			}
			if tt.assertOut != nil {
				tt.assertOut(t, out)
			}
		})
	}
}
