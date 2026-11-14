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
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"google.golang.org/protobuf/types/known/emptypb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

type providerLeaseCodec struct {
	closed atomic.Bool
	closes *atomic.Int64
}

func (*providerLeaseCodec) Name() string { return "provider-lease-json" }
func (c *providerLeaseCodec) Marshal(v any) ([]byte, error) {
	if c.closed.Load() {
		return nil, fmt.Errorf("codec already closed")
	}
	return json.Marshal(v)
}
func (c *providerLeaseCodec) Unmarshal(b []byte, v any) error {
	if c.closed.Load() {
		return fmt.Errorf("codec already closed")
	}
	return json.Unmarshal(b, v)
}
func (c *providerLeaseCodec) Close() error { c.closed.Store(true); c.closes.Add(1); return nil }

func TestDiscoveredProvidersShareCodecLease(t *testing.T) {
	var made, closed atomic.Int64
	name := "provider-lease-json"
	require.NoError(t, extension.RegisterSerialization(serialization.Registration{Protocol: "triple", Name: name, Mode: serialization.WrappedObjectMode, WireName: "proto", New: func(options serialization.Options) (serialization.Codec, error) {
		require.Equal(t, "LOCAL", options.Parameters["plugin-tag"])
		require.Equal(t, name, options.Parameters["serialization"])
		made.Add(1)
		return &providerLeaseCodec{closes: &closed}, nil
	}}))
	serverCodec := &providerLeaseCodec{closes: &atomic.Int64{}}
	_, opts := tri.ConfiguredSerializationOptions(serverCodec, map[string]tri.Codec{name: serverCodec}, map[string]tri.Codec{}, true, false)
	handler := tri.NewUnaryHandler("/review.Service/Echo", func() any { return []any{new(string)} }, func(_ context.Context, r *tri.Request) (*tri.Response, error) {
		return tri.NewResponse([]any{"hello " + *r.Any().([]any)[0].(*string)}), nil
	}, opts...)
	// Distinct HTTP endpoints model two independently discovered providers.
	aServer := httptest.NewServer(h2c.NewHandler(handler, &http2.Server{}))
	defer aServer.Close()
	bServer := httptest.NewServer(h2c.NewHandler(handler, &http2.Server{}))
	defer bServer.Close()
	ref := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithPath("review.Service"), common.WithParamsValue("serialization", name), common.WithParamsValue("plugin-tag", "LOCAL"))
	require.NoError(t, (&TripleProtocol{}).PrepareCodec(ref, "consumer"))
	providerURL := func(address string) *common.URL {
		u, err := common.NewURL("tri://" + strings.TrimPrefix(address, "http://") + "/review.Service")
		require.NoError(t, err)
		u.SetParam("serialization", "hessian2")
		u.SetParam("plugin-tag", "REMOTE")
		return u.MergeURL(ref)
	}
	a, err := NewTripleInvoker(providerURL(aServer.URL))
	require.NoError(t, err)
	defer a.Destroy()
	b, err := NewTripleInvoker(providerURL(bServer.URL))
	require.NoError(t, err)
	defer b.Destroy()
	call := func(inv *TripleInvoker) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var reply string
		require.NoError(t, inv.getClientManager().callUnary(ctx, "Echo", []any{"world"}, &reply, nil, nil))
		require.Equal(t, "hello world", reply)
	}
	call(a)
	call(b)
	a.Destroy()
	require.Zero(t, closed.Load())
	call(b)
	// A failing new provider relinquishes only its own lease.
	bad := providerURL(bServer.URL)
	bad.SetAttribute(constant.TLSConfigKey, "invalid")
	_, err = newClientManager(bad)
	require.Error(t, err)
	require.Zero(t, closed.Load())
	call(b)
	require.NoError(t, (&TripleProtocol{}).ReleaseCodec(ref, "consumer"))
	require.Zero(t, closed.Load(), "rollback cannot close a live provider's binding")
	b.Destroy()
	require.EqualValues(t, 1, closed.Load())
	require.EqualValues(t, 1, made.Load())
	// Discovery recovery after losing every provider creates a fresh generation.
	next, err := NewTripleInvoker(providerURL(bServer.URL))
	require.NoError(t, err)
	defer next.Destroy()
	call(next)
	require.EqualValues(t, 2, made.Load())
	next.Destroy()
	require.EqualValues(t, 2, closed.Load())
}

func TestProviderCodecConcurrentRelease(t *testing.T) {
	var closed atomic.Int64
	owner := &serialization.Resources{}
	owner.Add(&providerLeaseCodec{closes: &closed})
	binding := &serializationBinding{resources: owner}
	ref := common.NewURLWithOptions(common.WithProtocol("tri"))
	ref.SetAttribute("triple.serialization.consumer", binding)
	const providers = 24
	releases := make([]func() error, providers)
	for i := range releases {
		_, _, _, release, err := acquireClientSerialization(ref.Clone())
		require.NoError(t, err)
		releases[i] = release
	}
	var wg sync.WaitGroup
	for _, release := range releases {
		wg.Add(1)
		go func(release func() error) { defer wg.Done(); _ = release(); _ = release() }(release)
	}
	wg.Wait()
	require.EqualValues(t, 1, closed.Load())
	require.Zero(t, binding.users)
	require.True(t, binding.retired)
}

type codecEpochProbe struct {
	name          string
	entered, gate chan struct{}
	closes        atomic.Int64
}

func (c *codecEpochProbe) Name() string                    { return c.name }
func (c *codecEpochProbe) Marshal(v any) ([]byte, error)   { return json.Marshal(v) }
func (c *codecEpochProbe) Unmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }
func (c *codecEpochProbe) Close() error {
	c.closes.Add(1)
	if c.entered != nil {
		close(c.entered)
		<-c.gate
	}
	return nil
}
func TestCodecTripleGenerationInterleaveAndRetry(t *testing.T) {
	const name = "codec-triple-epochs"
	const accepted = "codec-triple-epochs-accepted"
	old := &codecEpochProbe{name: name, entered: make(chan struct{}), gate: make(chan struct{})}
	failed, next := &codecEpochProbe{name: name}, &codecEpochProbe{name: name}
	sentinel := errors.New("accepted codec construction failed")
	count := 0
	fail := false
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "triple", Name: name, Mode: serialization.WrappedObjectMode, WireName: "proto", New: func(o serialization.Options) (serialization.Codec, error) {
		require.Equal(t, "local", o.Parameters["plugin-tag"])
		count++
		switch count {
		case 1:
			return old, nil
		case 2:
			return failed, nil
		default:
			return next, nil
		}
	}})
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "triple", Name: accepted, Mode: serialization.WrappedObjectMode, WireName: "proto", New: func(serialization.Options) (serialization.Codec, error) {
		if fail {
			return nil, sentinel
		}
		return &codecEpochProbe{name: accepted}, nil
	}})
	ref := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithParamsValue("serialization", name), common.WithParamsValue("accepted-serializations", accepted), common.WithParamsValue("plugin-tag", "local"))
	require.NoError(t, (&TripleProtocol{}).PrepareCodec(ref, "consumer"))
	remote := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithParamsValue("serialization", "hessian2"), common.WithParamsValue("plugin-tag", "remote")).MergeURL(ref)
	_, _, _, release, err := acquireClientSerialization(remote)
	require.NoError(t, err)
	var once sync.Once
	defer once.Do(func() { close(old.gate) })
	done := make(chan struct{})
	go func() { _ = release(); close(done) }()
	select {
	case <-old.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("old generation Close not reached")
	}
	fail = true
	_, _, _, _, err = acquireClientSerialization(remote.Clone())
	require.ErrorIs(t, err, sentinel)
	require.EqualValues(t, 1, failed.closes.Load())
	fail = false
	_, _, owner, releaseNext, err := acquireClientSerialization(remote.Clone())
	require.NoError(t, err)
	require.True(t, owner.Begin())
	owner.End()
	require.Zero(t, next.closes.Load())
	once.Do(func() { close(old.gate) })
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("old generation did not close")
	}
	require.NoError(t, release())
	require.Zero(t, next.closes.Load())
	require.NoError(t, releaseNext())
	require.EqualValues(t, 1, next.closes.Load())
}

func TestCodecServerStopOwnsAllServiceCodecs(t *testing.T) {
	server := NewServer(global.DefaultTripleConfig())
	server.triServer = tri.NewServer("", global.DefaultTripleConfig())
	a, b := &codecEpochProbe{name: "a"}, &codecEpochProbe{name: "b"}
	first, second := &serialization.Resources{}, &serialization.Resources{}
	first.Add(a)
	second.Add(b)
	for i, owner := range []*serialization.Resources{first, second} {
		path := fmt.Sprintf("codec.Stop%d", i)
		url := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithPath(path), common.WithParamsValue("interface", path))
		url.SetAttribute("triple.serialization.resources.provider", owner)
		info := &common.ServiceInfo{Methods: []common.MethodInfo{{Name: "Call", Type: constant.CallUnary, ReqInitFunc: func() any { return &emptypb.Empty{} }}}}
		server.registerServiceHandlers(base.NewBaseInvoker(url), info, nil)
	}
	require.Len(t, server.codecResources, 2)
	// The same handler options protect decode, business code and response encode;
	// exercise a real local request with the resource registered on the Server.
	entered, gate := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(gate) })
	handler := tri.NewUnaryHandler("/codec.Stop/Call", func() any { return &emptypb.Empty{} }, func(context.Context, *tri.Request) (*tri.Response, error) {
		close(entered)
		<-gate
		return tri.NewResponse(&emptypb.Empty{}), nil
	}, tri.WithSerializationResources(first))
	local := httptest.NewUnstartedServer(handler)
	local.EnableHTTP2 = true
	local.StartTLS()
	defer func() { once.Do(func() { close(gate) }); local.Close() }()
	client := tri.NewClient(local.Client(), local.URL+"/codec.Stop")
	call := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		call <- client.CallUnary(ctx, tri.NewRequest(&emptypb.Empty{}), "Call", tri.NewResponse(&emptypb.Empty{}))
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("handler not entered")
	}
	done := make(chan struct{})
	go func() { server.Stop(); close(done) }()
	select {
	case <-done:
		t.Fatal("Stop released active handler")
	case <-time.After(20 * time.Millisecond):
	}
	require.Zero(t, a.closes.Load())
	require.Zero(t, b.closes.Load())
	once.Do(func() { close(gate) })
	select {
	case err := <-call:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("call stuck")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop stuck")
	}
	require.EqualValues(t, 1, a.closes.Load())
	require.EqualValues(t, 1, b.closes.Load())
	require.Error(t, client.CallUnary(context.Background(), tri.NewRequest(&emptypb.Empty{}), "Call", tri.NewResponse(&emptypb.Empty{})))
}
