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

package extension

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

type testSerialization struct {
	name       string
	parameters map[string]string
}

func (c *testSerialization) Name() string              { return c.name }
func (*testSerialization) Marshal(any) ([]byte, error) { return []byte("test"), nil }
func (*testSerialization) Unmarshal([]byte, any) error { return nil }

func TestSerializationConstruction(t *testing.T) {
	var calls atomic.Int64
	reg := serialization.Registration{Protocol: "triple", Name: "test-factory", Mode: serialization.WrappedObjectMode, WireName: "proto", New: func(o serialization.Options) (serialization.Codec, error) {
		calls.Add(1)
		o.Parameters["mutated"] = "factory"
		return &testSerialization{name: "test-factory", parameters: o.Parameters}, nil
	}}
	require.NoError(t, RegisterSerialization(reg))
	require.ErrorContains(t, RegisterSerialization(reg), "already registered")
	r, ok := LookupSerialization("tri", "test-factory")
	require.True(t, ok)
	parameters := map[string]string{"tag": "first"}
	a, err := BuildSerialization(r, serialization.Options{Parameters: parameters})
	require.NoError(t, err)
	parameters["tag"] = "second"
	b, err := BuildSerialization(r, serialization.Options{Parameters: parameters})
	require.NoError(t, err)
	require.Equal(t, "first", a.(*testSerialization).parameters["tag"])
	require.Equal(t, "second", b.(*testSerialization).parameters["tag"])
	require.NotContains(t, parameters, "mutated")
	require.EqualValues(t, 2, calls.Load())
	r.New = func(serialization.Options) (serialization.Codec, error) {
		return nil, errors.New("construction failed")
	}
	_, err = BuildSerialization(r, serialization.Options{})
	require.ErrorContains(t, err, "construction failed")
	r.New = func(serialization.Options) (serialization.Codec, error) { return (*testSerialization)(nil), nil }
	_, err = BuildSerialization(r, serialization.Options{})
	require.ErrorContains(t, err, "nil")
	r.New = func(serialization.Options) (serialization.Codec, error) {
		return &testSerialization{name: "wrong"}, nil
	}
	_, err = BuildSerialization(r, serialization.Options{})
	require.ErrorContains(t, err, "name")
}
func TestSerializationValidation(t *testing.T) {
	factory := func(serialization.Options) (serialization.Codec, error) { return &testSerialization{}, nil }
	for i, r := range []serialization.Registration{
		{Protocol: "tri", Name: "valid", Mode: serialization.MessageMode, WireName: "valid", New: factory},
		{Protocol: "triple", Name: "bad name", Mode: serialization.MessageMode, WireName: "bad name", New: factory},
		{Protocol: "triple", Name: "valid", Mode: serialization.MessageMode, WireName: "valid"},
		{Protocol: "triple", Name: "valid", Mode: serialization.StreamMode, New: factory},
		{Protocol: "grpc", Name: "valid", Mode: serialization.WrappedObjectMode, WireName: "proto", New: factory},
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) { require.Error(t, RegisterSerialization(r)) })
	}
	id := byte(31)
	r := serialization.Registration{Protocol: "dubbo", Name: "test-id", Mode: serialization.StreamMode, DubboID: &id, New: factory}
	require.NoError(t, RegisterSerialization(r))
	id = 0
	saved, ok := LookupSerialization("dubbo", "test-id")
	require.True(t, ok)
	require.Equal(t, byte(31), *saved.DubboID)
	*saved.DubboID = 29
	again, _ := LookupSerialization("dubbo", "test-id")
	require.Equal(t, byte(31), *again.DubboID)
	other := byte(31)
	r.Name = "test-id-conflict"
	r.DubboID = &other
	require.ErrorContains(t, RegisterSerialization(r), "already registered")
	other = 2
	require.ErrorContains(t, RegisterSerialization(r), "reserved")
	other = 0
	require.ErrorContains(t, RegisterSerialization(r), "5-bit")
}

type serializationSnapshotConfig struct {
	Values map[string]string `yaml:"values"`
	inits  *atomic.Int64
}

func (*serializationSnapshotConfig) Prefix() string { return "serialization-snapshot" }
func (c *serializationSnapshotConfig) New() Config {
	return &serializationSnapshotConfig{inits: c.inits}
}
func (c *serializationSnapshotConfig) Init(Scope) error         { c.inits.Add(1); return nil }
func (*serializationSnapshotConfig) FilterNames(Scope) []string { return nil }
func (c *serializationSnapshotConfig) CodecConfig() any {
	v := make(map[string]string)
	for k, x := range c.Values {
		v[k] = x
	}
	return v
}
func TestInitializeSerializationSnapshots(t *testing.T) {
	var calls atomic.Int64
	require.NoError(t, RegisterConfig(&serializationSnapshotConfig{inits: &calls}))
	defer UnregisterConfig("serialization-snapshot")
	raw := map[string]any{"serialization-snapshot": map[string]any{"consumer": map[string]any{"values": map[string]any{"tag": "first"}}}}
	a, err := InitializeWithResult(raw, nil, ClientScope)
	require.NoError(t, err)
	raw["serialization-snapshot"].(map[string]any)["consumer"].(map[string]any)["values"].(map[string]any)["tag"] = "second"
	b, err := InitializeWithResult(raw, nil, ClientScope)
	require.NoError(t, err)
	require.Equal(t, "first", a.CodecConfigs["serialization-snapshot"].(map[string]string)["tag"])
	require.Equal(t, "second", b.CodecConfigs["serialization-snapshot"].(map[string]string)["tag"])
	require.EqualValues(t, 2, calls.Load())
}

func TestLocalSerializationPolicySurvivesDiscovery(t *testing.T) {
	local := common.NewURLWithOptions(common.WithParamsValue("accepted-serializations", "private"))
	SnapshotSerializationPolicy(local)
	local.SetParam("accepted-serializations", "remote,other")
	names, explicit := AcceptedSerializations(local)
	require.True(t, explicit)
	require.Equal(t, []string{"private"}, names)
	names[0] = "changed"
	names, _ = AcceptedSerializations(local)
	require.Equal(t, "private", names[0])
	defaults := common.NewURLWithOptions()
	SnapshotSerializationPolicy(defaults)
	defaults.SetParam("accepted-serializations", "remote")
	_, explicit = AcceptedSerializations(defaults)
	require.False(t, explicit)
}

func TestCodecRegistrationValidationAndAtomicID(t *testing.T) {
	factory := func(serialization.Options) (serialization.Codec, error) { return &testSerialization{}, nil }
	bytep := func(v byte) *byte { return &v }
	for i, alter := range []func(*serialization.Registration){
		func(r *serialization.Registration) { r.Mode = 0 },
		func(r *serialization.Registration) { r.WireName = "" },
		func(r *serialization.Registration) { r.WireName = "different" },
		func(r *serialization.Registration) { r.Mode = serialization.WrappedObjectMode; r.WireName = "json" },
		func(r *serialization.Registration) {
			r.Protocol = "grpc"
			r.Mode = serialization.WrappedObjectMode
			r.WireName = "proto"
		},
		func(r *serialization.Registration) { r.DubboID = bytep(7) },
		func(r *serialization.Registration) {
			r.Protocol = "dubbo"
			r.Mode = serialization.StreamMode
			r.DubboID = nil
		},
		func(r *serialization.Registration) {
			r.Protocol = "dubbo"
			r.Mode = serialization.StreamMode
			r.DubboID = bytep(32)
		},
		func(r *serialization.Registration) {
			r.Protocol = "dubbo"
			r.Mode = serialization.StreamMode
			r.DubboID = bytep(21)
		},
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			name := fmt.Sprintf("codec-validate-%d", i)
			valid := serialization.Registration{Protocol: "triple", Name: name, WireName: name, Mode: serialization.MessageMode, New: factory}
			bad := valid
			alter(&bad)
			require.Error(t, RegisterSerialization(bad))
			_, exists := LookupSerialization(bad.Protocol, name)
			require.False(t, exists)
			require.NoError(t, RegisterSerialization(valid))
		})
	}
	// Failure after receiving an otherwise usable ID must not reserve either key.
	id := byte(26)
	reg := serialization.Registration{Protocol: "dubbo", Name: "codec-validation-id", Mode: serialization.MessageMode, DubboID: &id, New: factory}
	require.Error(t, RegisterSerialization(reg))
	reg.Mode = serialization.StreamMode
	require.NoError(t, RegisterSerialization(reg))
	id = 25
	start := make(chan struct{})
	results := make(chan struct {
		name string
		err  error
	}, 2)
	for _, name := range []string{"codec-id-racer-a", "codec-id-racer-b"} {
		go func(name string) {
			<-start
			err := RegisterSerialization(serialization.Registration{Protocol: "dubbo", Name: name, Mode: serialization.StreamMode, DubboID: &id, New: factory})
			results <- struct {
				name string
				err  error
			}{name, err}
		}(name)
	}
	close(start)
	wins := 0
	loser := ""
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err == nil {
			wins++
		} else {
			loser = r.name
			_, exists := LookupSerialization("dubbo", loser)
			require.False(t, exists)
		}
	}
	require.Equal(t, 1, wins)
	retryID := byte(24)
	require.NoError(t, RegisterSerialization(serialization.Registration{Protocol: "dubbo", Name: loser, Mode: serialization.StreamMode, DubboID: &retryID, New: factory}))
}

type codecConstructionProbe struct {
	testSerialization
	closes int
}

func (c *codecConstructionProbe) Close() error { c.closes++; return nil }
func TestCodecSerializationFactoryCleanup(t *testing.T) {
	sentinel := errors.New("construction sentinel")
	for _, test := range []struct {
		name   string
		mode   serialization.Mode
		result func(*codecConstructionProbe) (serialization.Codec, error)
		bad    bool
	}{
		{"nil", serialization.MessageMode, func(*codecConstructionProbe) (serialization.Codec, error) { return nil, nil }, true},
		{"typed-nil", serialization.MessageMode, func(*codecConstructionProbe) (serialization.Codec, error) { return (*codecConstructionProbe)(nil), nil }, true},
		{"error", serialization.MessageMode, func(*codecConstructionProbe) (serialization.Codec, error) { return nil, sentinel }, true},
		{"instance-error", serialization.MessageMode, func(c *codecConstructionProbe) (serialization.Codec, error) { return c, sentinel }, true},
		{"wrong-name", serialization.MessageMode, func(c *codecConstructionProbe) (serialization.Codec, error) { c.name = "wrong"; return c, nil }, true},
		{"missing-stream", serialization.StreamMode, func(c *codecConstructionProbe) (serialization.Codec, error) { return c, nil }, true},
		{"success", serialization.MessageMode, func(c *codecConstructionProbe) (serialization.Codec, error) { return c, nil }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := &codecConstructionProbe{testSerialization: testSerialization{name: "codec-build"}}
			var factoryErr error
			var returned serialization.Codec
			reg := serialization.Registration{Protocol: "triple", Name: "codec-build", Mode: test.mode, New: func(serialization.Options) (serialization.Codec, error) {
				returned, factoryErr = test.result(c)
				return returned, factoryErr
			}}
			got, err := BuildSerialization(reg, serialization.Options{})
			if test.bad {
				require.Error(t, err)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Same(t, c, got)
			}
			if factoryErr != nil {
				require.ErrorIs(t, err, sentinel)
			}
			expected := 0
			if test.bad && !serializationCodecIsNil(returned) {
				expected = 1
			}
			require.Equal(t, expected, c.closes)
		})
	}
}

type codecFinalConfig struct {
	Value     string `yaml:"value"`
	prefix    string
	fail      error
	snapshots *atomic.Int64
}

func (c *codecFinalConfig) Prefix() string { return c.prefix }
func (c *codecFinalConfig) New() Config {
	return &codecFinalConfig{prefix: c.prefix, fail: c.fail, snapshots: c.snapshots}
}
func (c *codecFinalConfig) Init(scope Scope) error {
	c.Value += "/initialized/" + fmt.Sprint(scope)
	return c.fail
}
func (*codecFinalConfig) FilterNames(Scope) []string { return nil }
func (c *codecFinalConfig) CodecConfig() any         { c.snapshots.Add(1); return c.Value }
func TestCodecConfigurationFinalValuesAndAtomicPublication(t *testing.T) {
	var snapshots atomic.Int64
	const first = "codec-config-a"
	const last = "codec-config-z"
	sentinel := errors.New("last init failed")
	a, z := &codecFinalConfig{prefix: first, snapshots: &snapshots}, &codecFinalConfig{prefix: last, fail: sentinel, snapshots: &snapshots}
	require.NoError(t, RegisterConfig(a))
	t.Cleanup(func() { UnregisterConfig(first) })
	require.NoError(t, RegisterConfig(z))
	t.Cleanup(func() { UnregisterConfig(last) })
	raw := map[string]any{first: map[string]any{"consumer": map[string]any{"value": "consumer"}, "provider": map[string]any{"value": "provider"}}, last: map[string]any{"consumer": map[string]any{"value": "failure"}}}
	failed, err := InitializeWithResult(raw, nil, ClientScope)
	require.ErrorIs(t, err, sentinel)
	require.Empty(t, failed.CodecConfigs)
	require.Zero(t, snapshots.Load())
	delete(raw, last)
	for _, scope := range []Scope{ClientScope, ServerScope} {
		result, err := InitializeWithResult(raw, nil, scope)
		require.NoError(t, err)
		value := "consumer"
		if scope == ServerScope {
			value = "provider"
		}
		require.Equal(t, value+"/initialized/"+fmt.Sprint(scope), result.CodecConfigs[first])
	}
	require.EqualValues(t, 2, snapshots.Load())
}
