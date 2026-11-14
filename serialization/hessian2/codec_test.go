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

package hessian2

import (
	"bytes"
	"reflect"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

func TestStreamOwnershipAndRecovery(t *testing.T) {
	c := &Codec{}
	encoder, err := c.NewEncoder()
	require.NoError(t, err)
	require.NoError(t, encoder.Encode("first"))
	first, err := encoder.Finish()
	require.NoError(t, err)
	saved := bytes.Clone(first)
	require.NoError(t, encoder.Encode("second"))
	_, err = encoder.Finish()
	require.NoError(t, err)
	require.Equal(t, saved, first)
	require.NoError(t, encoder.Close())
	require.Error(t, encoder.Encode("closed"))
	require.Error(t, c.Unmarshal([]byte{255}, new(string)))
	data, err := c.Marshal("valid")
	require.NoError(t, err)
	var got string
	require.NoError(t, c.Unmarshal(data, &got))
	require.Equal(t, "valid", got)
	require.Error(t, c.Unmarshal(data, got))
	require.Error(t, c.Unmarshal(data, new(int)))
	var wg sync.WaitGroup
	for range 24 {
		wg.Go(func() {
			b, e := c.Marshal("parallel")
			if e != nil {
				t.Error(e)
				return
			}
			var v string
			if e = c.Unmarshal(b, &v); e != nil {
				t.Error(e)
			} else if v != "parallel" {
				t.Errorf("bad result %s", v)
			}
		})
	}
	wg.Wait()
}

func TestStreamReferenceScope(t *testing.T) {
	c := &Codec{}
	shared := map[string]string{"key": "value"}
	e, err := c.NewEncoder()
	require.NoError(t, err)
	defer e.Close()
	require.NoError(t, e.Encode(shared))
	require.NoError(t, e.Encode(shared))
	data, err := e.Finish()
	require.NoError(t, err)
	d, err := c.NewDecoder(data)
	require.NoError(t, err)
	defer d.Close()
	var first, second any
	require.NoError(t, d.Decode(&first))
	require.NoError(t, d.Decode(&second))
	require.Equal(t, reflect.Map, reflect.TypeOf(first).Kind())
	require.Equal(t, reflect.ValueOf(first).Pointer(), reflect.ValueOf(second).Pointer())
	standalone, err := c.Marshal(shared)
	require.NoError(t, err)
	var independent any
	require.NoError(t, c.Unmarshal(standalone, &independent))
	require.NotEqual(t, reflect.ValueOf(first).Pointer(), reflect.ValueOf(independent).Pointer())
}

func TestCodecHessianAttachmentTargetsAndNull(t *testing.T) {
	c := &Codec{}
	for _, test := range []struct {
		name  string
		value any
		want  map[string]any
		bad   bool
	}{
		{"string-keys", map[any]any{"tag": "value"}, map[string]any{"tag": "value"}, false},
		{"nonstring-key", map[any]any{int32(1): "value"}, nil, true},
		{"legacy-empty", "", nil, false},
		{"null", nil, nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			b, err := c.Marshal(test.value)
			require.NoError(t, err)
			got := map[string]any{"old": true}
			err = c.Unmarshal(b, &got)
			if test.bad {
				require.ErrorContains(t, err, "not a string")
			} else {
				require.NoError(t, err)
				require.Equal(t, test.want, got)
			}
		})
	}
	b, err := c.Marshal(nil)
	require.NoError(t, err)
	value := "old"
	require.NoError(t, c.Unmarshal(b, &value))
	require.Empty(t, value)
}
