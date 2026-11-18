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

package triple

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

type exportGenerationCodec struct {
	closed atomic.Bool
	closes *atomic.Int64
}

func (*exportGenerationCodec) Name() string { return "export-generation" }
func (c *exportGenerationCodec) Marshal(v any) ([]byte, error) {
	if c.closed.Load() {
		return nil, errors.New("closed generation")
	}
	return json.Marshal(v)
}
func (c *exportGenerationCodec) Unmarshal(b []byte, v any) error {
	if c.closed.Load() {
		return errors.New("closed generation")
	}
	return json.Unmarshal(b, v)
}
func (c *exportGenerationCodec) Close() error { c.closed.Store(true); c.closes.Add(1); return nil }

type exportGenerationInvoker struct{ *base.BaseInvoker }

func (c *exportGenerationInvoker) Invoke(context.Context, base.Invocation) result.Result {
	r := &result.RPCResult{}
	if c.IsDestroyed() {
		r.SetError(errors.New("invoker-destroyed"))
	} else {
		r.SetResult("hello")
	}
	return r
}
func TestProviderExportGenerationsSurviveClonedURLs(t *testing.T) {
	var made, closed atomic.Int64
	var settings []string
	require.NoError(t, extension.RegisterSerialization(serialization.Registration{Protocol: "triple", Name: "export-generation", Mode: serialization.WrappedObjectMode, WireName: "proto", New: func(o serialization.Options) (serialization.Codec, error) {
		if o.Parameters["epoch"] == "fail" {
			return nil, errors.New("factory failed")
		}
		made.Add(1)
		settings = append(settings, o.Parameters["epoch"])
		return &exportGenerationCodec{closes: &closed}, nil
	}}))
	for _, overlap := range []bool{false, true} {
		t.Run(strconv.FormatBool(overlap), func(t *testing.T) {
			tp := NewTripleProtocol()
			defer tp.Destroy()
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			address := listener.Addr().String()
			require.NoError(t, listener.Close())
			url, err := common.NewURL("tri://"+address+"/codec.ExportGeneration", common.WithParamsValue("interface", "codec.ExportGeneration"), common.WithParamsValue("serialization", "export-generation"), common.WithParamsValue("epoch", "first"))
			require.NoError(t, err)
			url.SetAttribute(constant.ServiceInfoKey, &common.ServiceInfo{Methods: []common.MethodInfo{{Name: "Echo", Type: constant.CallUnary, ReqInitFunc: func() any { return []any{new(string)} }}}})
			require.NoError(t, tp.PrepareCodec(url, "provider"))
			first := tp.Export(&exportGenerationInvoker{base.NewBaseInvoker(url)})
			waitExportListener(t, address)
			// Registry override URLs are cloned before the old export is retired.
			next := url.Clone()
			next.SetParam("epoch", "second")
			old, _ := url.GetAttribute("triple.serialization.resources.provider")
			clientOpts, _ := tri.ConfiguredSerializationOptions(&exportGenerationCodec{closes: &atomic.Int64{}}, nil, nil, true, false)
			cli := tri.NewClient(http.DefaultClient, "http://"+address+"/codec.ExportGeneration", clientOpts...)
			call := func() {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				var reply string
				require.NoError(t, cli.CallUnary(ctx, tri.NewRequest([]any{"request"}), "Echo", tri.NewResponse(&reply)))
				require.Equal(t, "hello", reply)
			}
			call()
			broken := url.Clone()
			broken.SetParam("epoch", "fail")
			require.Panics(t, func() { tp.Export(&exportGenerationInvoker{base.NewBaseInvoker(broken)}) })
			current, ok := tp.ExporterMap().Load(url.ServiceKey())
			require.True(t, ok)
			require.Same(t, first, current)
			call()
			before := closed.Load()
			if !overlap {
				first.UnExport()
				require.Equal(t, before+1, closed.Load())
			}
			second := tp.Export(&exportGenerationInvoker{base.NewBaseInvoker(next)})
			fresh, _ := next.GetAttribute("triple.serialization.resources.provider")
			require.NotSame(t, old, fresh)
			call()
			first.UnExport()
			call()
			second.UnExport()
			require.Equal(t, before+2, closed.Load())
		})
	}
	require.EqualValues(t, 4, made.Load())
	require.Equal(t, []string{"first", "second", "first", "second"}, settings)
}
func TestProviderExportGenerationFailurePreservesPreviousBinding(t *testing.T) {
	const name = "export-generation-failure"
	var fail atomic.Bool
	require.NoError(t, extension.RegisterSerialization(serialization.Registration{Protocol: "triple", Name: name, Mode: serialization.WrappedObjectMode, WireName: "proto", New: func(serialization.Options) (serialization.Codec, error) {
		if fail.Load() {
			return nil, errors.New("factory failed")
		}
		return &codecEpochProbe{name: name}, nil
	}}))
	tp := &TripleProtocol{}
	url := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithParamsValue("serialization", name))
	require.NoError(t, tp.prepareProviderExport(url))
	raw, _ := url.GetAttribute("triple.serialization.provider")
	old := raw.(*serializationBinding)
	next := url.Clone()
	fail.Store(true)
	require.ErrorContains(t, tp.prepareProviderExport(next), "factory failed")
	still, _ := next.GetAttribute("triple.serialization.provider")
	require.Same(t, old, still)
	require.True(t, old.resources.Begin())
	old.resources.End()
	fail.Store(false)
	require.NoError(t, tp.prepareProviderExport(next))
	fresh, _ := next.GetAttribute("triple.serialization.provider")
	require.NotSame(t, old, fresh)
	require.NoError(t, tp.ReleaseCodec(url, "provider"))
	require.NoError(t, tp.ReleaseCodec(next, "provider"))
	// A prepared binding released before Export must also be rebuilt.
	unused := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithParamsValue("serialization", name))
	require.NoError(t, tp.PrepareCodec(unused, "provider"))
	retired, _ := unused.GetAttribute("triple.serialization.provider")
	require.NoError(t, tp.ReleaseCodec(unused, "provider"))
	require.NoError(t, tp.prepareProviderExport(unused))
	revived, _ := unused.GetAttribute("triple.serialization.provider")
	require.NotSame(t, retired, revived)
	require.NoError(t, tp.ReleaseCodec(unused, "provider"))
}

func waitExportListener(t *testing.T, address string) {
	t.Helper()
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", address, 50*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 3*time.Second, time.Millisecond)
}
