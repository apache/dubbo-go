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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"encoding/json"
	"errors"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/mem"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
	grpcgo "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/serialization"
)

type testGRPCCodec struct{ calls *atomic.Int64 }

func (testGRPCCodec) Name() string                      { return "unit-grpc-json" }
func (c testGRPCCodec) Marshal(v any) ([]byte, error)   { c.calls.Add(1); return json.Marshal(v) }
func (c testGRPCCodec) Unmarshal(b []byte, v any) error { c.calls.Add(1); return json.Unmarshal(b, v) }
func TestGRPCSerializationRPCAndConflict(t *testing.T) {
	var calls, constructions atomic.Int64
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "grpc", Name: "unit-grpc-json", Mode: serialization.MessageMode, WireName: "unit-grpc-json", ConfigPrefix: "grpc-test", New: func(o serialization.Options) (serialization.Codec, error) {
		constructions.Add(1)
		require.Equal(t, "", o.Side)
		return testGRPCCodec{calls: &calls}, nil
	}})
	url := common.NewURLWithOptions(common.WithProtocol("grpc"), common.WithParamsValue("serialization", "unit-grpc-json"), common.WithAttribute(extension.SerializationConfigAttribute, map[string]any{"grpc-test": "first"}))
	gp := NewGRPCProtocol()
	require.NoError(t, gp.PrepareCodec(url, "provider"))
	require.NoError(t, gp.PrepareCodec(url.Clone(), "consumer"))
	require.EqualValues(t, 1, constructions.Load())
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
	require.EqualValues(t, 4, calls.Load())
	// Standard protobuf health requests remain available on the same server.
	response, err = client.Check(ctx, &healthpb.HealthCheckRequest{Service: "echo"})
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, response.Status)
	require.EqualValues(t, 4, calls.Load())
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
		extension.MustRegisterSerialization(serialization.Registration{Protocol: "grpc", Name: name, Mode: serialization.MessageMode, WireName: name, New: func(serialization.Options) (serialization.Codec, error) {
			t.Error("conflicting factory was called")
			return codecGRPCProbe{name: name, calls: &atomic.Int64{}}, nil
		}})
		if version == "v1" {
			encoding.RegisterCodec(codecGRPCProbe{name: name, calls: &atomic.Int64{}})
		} else {
			encoding.RegisterCodecV2(codecExternalV2{name})
		}
		require.ErrorContains(t, gp.PrepareCodec(common.NewURLWithOptions(common.WithProtocol("grpc"), common.WithParamsValue("serialization", name)), "consumer"), "outside")
	}
	const name = "codec-grpc-retry"
	sentinel := errors.New("grpc factory failed")
	count := 0
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "grpc", Name: name, Mode: serialization.MessageMode, WireName: name, ConfigPrefix: "codec-grpc-prefix", New: func(o serialization.Options) (serialization.Codec, error) {
		count++
		if count == 1 {
			return nil, sentinel
		}
		require.Equal(t, "local", o.Parameters["codec-grpc-prefix.tag"])
		return codecGRPCProbe{name: name, calls: &atomic.Int64{}}, nil
	}})
	url := common.NewURLWithOptions(common.WithProtocol("grpc"), common.WithParamsValue("serialization", name), common.WithParamsValue("codec-grpc-prefix.tag", "local"))
	require.ErrorIs(t, gp.PrepareCodec(url, "consumer"), sentinel)
	require.Nil(t, encoding.GetCodec(name))
	require.NoError(t, gp.PrepareCodec(url, "consumer"))
	require.Equal(t, 2, count)
	conflict := url.Clone()
	conflict.SetParam("codec-grpc-prefix.tag", "changed")
	require.ErrorContains(t, gp.PrepareCodec(conflict, "provider"), "different configuration")
	require.Equal(t, 2, count)
}

type codecHealthStub struct{}

func (codecHealthStub) GetDubboStub(conn *grpcgo.ClientConn) healthpb.HealthClient {
	return healthpb.NewHealthClient(conn)
}
func TestCodecPublicGRPCClientDefaultSubtype(t *testing.T) {
	const name = "codec-grpc-default-subtype"
	var calls atomic.Int64
	extension.MustRegisterSerialization(serialization.Registration{Protocol: "grpc", Name: name, Mode: serialization.MessageMode, WireName: name, New: func(serialization.Options) (serialization.Codec, error) {
		return codecGRPCProbe{name: name, calls: &calls}, nil
	}})
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
	require.EqualValues(t, 4, calls.Load())
}
