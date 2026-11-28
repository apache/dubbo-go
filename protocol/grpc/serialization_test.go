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
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"

	grpcgo "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/mem"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

type testGRPCCodec struct{ calls *atomic.Int64 }

func (testGRPCCodec) Name() string                      { return "unit-grpc-json" }
func (c testGRPCCodec) Marshal(v any) ([]byte, error)   { c.calls.Add(1); return json.Marshal(v) }
func (c testGRPCCodec) Unmarshal(b []byte, v any) error { c.calls.Add(1); return json.Unmarshal(b, v) }

var grpcTestCalls, grpcTestConstructions, grpcPublicCalls atomic.Int64
var grpcChoiceCalls atomic.Int64
var grpcTestOptions serialization.Options
var grpcMergedOptions serialization.Options
var grpcRetryOptions serialization.Options
var grpcRetryCount atomic.Int64
var errGRPCFactoryFailure = errors.New("grpc factory failed")

func init() {
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "grpc", Name: "unit-grpc-json", Mode: serialization.MessageMode, WireName: "unit-grpc-json", ConfigPrefix: "grpc-test", New: func(o serialization.Options) (serialization.Codec, error) {
		grpcTestConstructions.Add(1)
		grpcTestOptions = o
		return testGRPCCodec{calls: &grpcTestCalls}, nil
	}})
	for _, version := range []string{"v1", "v2"} {
		name := "codec-grpc-external-" + version
		extension.MustRegisterSerialization(serialization.Registration{Protocol: "grpc", Name: name, Mode: serialization.MessageMode, WireName: name, New: func(serialization.Options) (serialization.Codec, error) {
			return nil, errors.New("conflicting factory was called")
		}})
		if version == "v1" {
			encoding.RegisterCodec(codecGRPCProbe{name: name, calls: &atomic.Int64{}})
		} else {
			encoding.RegisterCodecV2(codecExternalV2{name})
		}
	}
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "grpc", Name: "codec-grpc-retry", Mode: serialization.MessageMode, WireName: "codec-grpc-retry", ConfigPrefix: "codec-grpc-prefix", New: func(o serialization.Options) (serialization.Codec, error) {
		if grpcRetryCount.Add(1) == 1 {
			return nil, errGRPCFactoryFailure
		}
		grpcRetryOptions = o
		return codecGRPCProbe{name: "codec-grpc-retry", calls: &atomic.Int64{}}, nil
	}})
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "grpc", Name: "codec-grpc-default-subtype", Mode: serialization.MessageMode, WireName: "codec-grpc-default-subtype", New: func(serialization.Options) (serialization.Codec, error) {
		return codecGRPCProbe{name: "codec-grpc-default-subtype", calls: &grpcPublicCalls}, nil
	}})
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "grpc", Name: "codec-grpc-during-rpc", Mode: serialization.MessageMode, WireName: "codec-grpc-during-rpc", New: func(serialization.Options) (serialization.Codec, error) {
		return codecGRPCProbe{name: "codec-grpc-during-rpc", calls: &atomic.Int64{}}, nil
	}})
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "grpc", Name: "codec-grpc-consumer-choice", Mode: serialization.MessageMode, WireName: "codec-grpc-consumer-choice", New: func(serialization.Options) (serialization.Codec, error) {
		return codecGRPCProbe{name: "codec-grpc-consumer-choice", calls: &grpcChoiceCalls}, nil
	}})
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "grpc", Name: "codec-grpc-merged-config", Mode: serialization.MessageMode, WireName: "codec-grpc-merged-config", ConfigPrefix: "codec-grpc-merged", New: func(o serialization.Options) (serialization.Codec, error) {
		grpcMergedOptions = o
		return codecGRPCProbe{name: "codec-grpc-merged-config", calls: &atomic.Int64{}}, nil
	}})
}

func TestGRPCSerializationRPCAndConflict(t *testing.T) {
	url := common.NewURLWithOptions(common.WithProtocol("grpc"), common.WithParamsValue("serialization", "unit-grpc-json"), common.WithAttribute(extension.SerializationConfigAttribute, map[string]any{"grpc-test": "first"}))
	gp := NewGRPCProtocol()
	require.NoError(t, gp.PrepareCodec(url, "provider"))
	require.NoError(t, gp.PrepareCodec(url.Clone(), "consumer"))
	require.EqualValues(t, 1, grpcTestConstructions.Load())
	require.Empty(t, grpcTestOptions.Side)
	conflict := url.Clone()
	conflict.SetAttribute(extension.SerializationConfigAttribute, map[string]any{"grpc-test": "second"})
	require.ErrorContains(t, gp.PrepareCodec(conflict, "provider"), "different configuration")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpcgo.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("echo", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	go server.Serve(listener)
	defer server.Stop()
	conn, err := grpcgo.NewClient(listener.Addr().String(), grpcgo.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	client := healthpb.NewHealthClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := client.Check(ctx, &healthpb.HealthCheckRequest{Service: "echo"}, grpcgo.CallContentSubtype("unit-grpc-json"))
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, response.Status)
	require.EqualValues(t, 4, grpcTestCalls.Load())
	// Standard protobuf health requests remain available on the same server.
	response, err = client.Check(ctx, &healthpb.HealthCheckRequest{Service: "echo"})
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, response.Status)
	require.EqualValues(t, 4, grpcTestCalls.Load())
}
func TestLegacyGRPCJSONSemantics(t *testing.T) {
	reg, ok := extension.LookupSerialization("grpc", "json")
	require.True(t, ok)
	codec, err := extension.BuildSerialization(reg, serialization.Options{})
	require.NoError(t, err)
	data, err := codec.Marshal(&healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	require.JSONEq(t, `{"service":""}`, string(data))
	data, err = codec.Marshal(&healthpb.HealthCheckResponse{})
	require.NoError(t, err)
	require.JSONEq(t, `{"status":"UNKNOWN"}`, string(data))
}

type codecGRPCProbe struct {
	name  string
	calls *atomic.Int64
}

func (c codecGRPCProbe) Name() string                    { return c.name }
func (c codecGRPCProbe) Marshal(v any) ([]byte, error)   { c.calls.Add(1); return json.Marshal(v) }
func (c codecGRPCProbe) Unmarshal(b []byte, v any) error { c.calls.Add(1); return json.Unmarshal(b, v) }

type codecExternalV2 struct{ name string }

func (c codecExternalV2) Name() string                       { return c.name }
func (codecExternalV2) Marshal(any) (mem.BufferSlice, error) { return nil, nil }
func (codecExternalV2) Unmarshal(mem.BufferSlice, any) error { return nil }
func TestCodecGRPCConflictsAndRetry(t *testing.T) {
	gp := NewGRPCProtocol()
	for _, version := range []string{"v1", "v2"} {
		name := "codec-grpc-external-" + version
		require.ErrorContains(t, gp.PrepareCodec(common.NewURLWithOptions(common.WithProtocol("grpc"), common.WithParamsValue("serialization", name)), "consumer"), "outside")
	}
	const name = "codec-grpc-retry"
	url := common.NewURLWithOptions(common.WithProtocol("grpc"), common.WithParamsValue("serialization", name), common.WithParamsValue("codec-grpc-prefix.tag", "local"))
	adapter := encoding.GetCodec(name)
	require.NotNil(t, adapter)
	require.ErrorContains(t, adapter.Unmarshal(nil, &healthpb.HealthCheckRequest{}), "not configured")
	require.ErrorIs(t, gp.PrepareCodec(url, "consumer"), errGRPCFactoryFailure)
	require.Same(t, adapter, encoding.GetCodec(name))
	require.ErrorContains(t, adapter.Unmarshal(nil, &healthpb.HealthCheckRequest{}), "not configured")
	require.NoError(t, gp.PrepareCodec(url, "consumer"))
	require.EqualValues(t, 2, grpcRetryCount.Load())
	require.Equal(t, "local", grpcRetryOptions.Parameters["codec-grpc-prefix.tag"])
	conflict := url.Clone()
	conflict.SetParam("codec-grpc-prefix.tag", "changed")
	require.ErrorContains(t, gp.PrepareCodec(conflict, "provider"), "different configuration")
	require.EqualValues(t, 2, grpcRetryCount.Load())
}

type codecHealthStub struct{}

func (codecHealthStub) GetDubboStub(conn *grpcgo.ClientConn) healthpb.HealthClient {
	return healthpb.NewHealthClient(conn)
}
func TestCodecPublicGRPCClientDefaultSubtype(t *testing.T) {
	const name = "codec-grpc-default-subtype"
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpcgo.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("codec", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	go server.Serve(listener)
	defer server.Stop()
	url, err := common.NewURL("grpc://"+listener.Addr().String()+"/grpc.health.v1.Health", common.WithParamsValue("serialization", name))
	require.NoError(t, err)
	url.SetAttribute(constant.RpcServiceKey, codecHealthStub{})
	client, err := NewClient(url)
	require.NoError(t, err)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	response, err := healthpb.NewHealthClient(client.ClientConn).Check(ctx, &healthpb.HealthCheckRequest{Service: "codec"})
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, response.Status)
	require.EqualValues(t, 4, grpcPublicCalls.Load())
}

func TestGRPCConsumerSubtypeSurvivesDiscoveryMerge(t *testing.T) {
	const name = "codec-grpc-consumer-choice"
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpcgo.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("codec", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	go server.Serve(listener)
	defer server.Stop()
	local := common.NewURLWithOptions(common.WithProtocol("grpc"), common.WithParamsValue("serialization", name))
	require.NoError(t, extension.PrepareCodecURL(local, "consumer"))
	provider, err := common.NewURL("grpc://"+listener.Addr().String()+"/grpc.health.v1.Health", common.WithParamsValue("serialization", "proto"))
	require.NoError(t, err)
	merged := provider.MergeURL(local)
	require.Equal(t, "proto", merged.GetParam("serialization", ""))
	merged.SetAttribute(constant.RpcServiceKey, codecHealthStub{})
	before := grpcChoiceCalls.Load()
	client, err := NewClient(merged)
	require.NoError(t, err)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	response, err := healthpb.NewHealthClient(client.ClientConn).Check(ctx, &healthpb.HealthCheckRequest{Service: "codec"})
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, response.Status)
	require.Greater(t, grpcChoiceCalls.Load(), before)
}

func TestGRPCConsumerConfigSurvivesProviderMerge(t *testing.T) {
	const name = "codec-grpc-merged-config"
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpcgo.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("codec", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	go server.Serve(listener)
	defer server.Stop()
	local := common.NewURLWithOptions(common.WithProtocol("grpc"), common.WithParamsValue("serialization", name), common.WithParamsValue("codec-grpc-merged.tag", "consumer"))
	require.NoError(t, extension.PrepareCodecURL(local, "consumer"))
	require.Equal(t, "consumer", grpcMergedOptions.Parameters["codec-grpc-merged.tag"])
	provider, err := common.NewURL("grpc://"+listener.Addr().String()+"/grpc.health.v1.Health", common.WithParamsValue("serialization", name), common.WithParamsValue("codec-grpc-merged.tag", "provider"))
	require.NoError(t, err)
	merged := provider.MergeURL(local)
	require.Equal(t, "provider", merged.GetParam("codec-grpc-merged.tag", ""))
	merged.SetAttribute(constant.RpcServiceKey, codecHealthStub{})
	client, err := NewClient(merged)
	require.NoError(t, err)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	response, err := healthpb.NewHealthClient(client.ClientConn).Check(ctx, &healthpb.HealthCheckRequest{Service: "codec"})
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, response.Status)
	require.Equal(t, "consumer", grpcMergedOptions.Parameters["codec-grpc-merged.tag"])
}

// Configuring another subtype must not write grpc-go's registry while an
// existing protobuf connection performs RPCs and looks up its own codec.
func TestGRPCSerializationPreparationDuringRPC(t *testing.T) {
	const name = "codec-grpc-during-rpc"
	adapter := encoding.GetCodec(name)
	require.NotNil(t, adapter)
	require.ErrorContains(t, adapter.Unmarshal(nil, &healthpb.HealthCheckRequest{}), "not configured")
	_, err := adapter.Marshal(&healthpb.HealthCheckRequest{})
	require.ErrorContains(t, err, "not configured")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpcgo.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("echo", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	go server.Serve(listener)
	defer server.Stop()
	conn, err := grpcgo.NewClient(listener.Addr().String(), grpcgo.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	client := healthpb.NewHealthClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = client.Check(ctx, &healthpb.HealthCheckRequest{Service: "echo"})
	require.NoError(t, err)
	start := make(chan struct{})
	var workers sync.WaitGroup
	failures := make(chan error, 2)
	workers.Add(2)
	go func() {
		defer workers.Done()
		<-start
		for range 1000 {
			if encoding.GetCodec("json") == nil || encoding.GetCodec(name) != adapter {
				failures <- errors.New("grpc registry changed during preparation")
				return
			}
		}
	}()
	go func() {
		defer workers.Done()
		<-start
		for range 20 {
			if _, callErr := client.Check(ctx, &healthpb.HealthCheckRequest{Service: "echo"}); callErr != nil {
				failures <- callErr
				return
			}
		}
	}()
	close(start)
	url := common.NewURLWithOptions(common.WithProtocol("grpc"), common.WithParamsValue("serialization", name))
	prepareErr := NewGRPCProtocol().PrepareCodec(url, "provider")
	workers.Wait()
	close(failures)
	for failure := range failures {
		require.NoError(t, failure)
	}
	require.NoError(t, prepareErr)
	require.Same(t, adapter, encoding.GetCodec(name))
	response, err := client.Check(ctx, &healthpb.HealthCheckRequest{Service: "echo"}, grpcgo.CallContentSubtype(name))
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, response.Status)
}

func TestGRPCLateDescriptorRegistrationRejected(t *testing.T) {
	gp := NewGRPCProtocol()
	require.NoError(t, gp.PrepareCodec(common.NewURLWithOptions(common.WithProtocol("grpc")), "consumer"))
	const name = "codec-grpc-too-late"
	err := extension.RegisterSerialization(serialization.Registration{Protocol: "grpc", Name: name, Mode: serialization.MessageMode, WireName: name, New: func(serialization.Options) (serialization.Codec, error) {
		t.Fatal("late factory must not be called")
		return nil, nil
	}})
	require.ErrorContains(t, err, "during init")
	_, found := extension.LookupSerialization("grpc", name)
	require.False(t, found)
	require.Nil(t, encoding.GetCodec(name))
}
