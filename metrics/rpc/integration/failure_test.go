//go:build rpc_failure_integration

/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package integration_test

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

import (
	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/getty"
)

// This test uses the production Filter, event bus, RPC Collector, Prometheus
// registry, DubboInvoker, ExchangeClient, Getty TCP client and Hessian codec.
// The peer only controls wire responses; it does not synthesize metric events.
// Opt in separately: Getty v1.4.10 has known connection-teardown races, also
// reproduced by baseline tests already excluded in the root Makefile race run.
func TestRPCFailureAttributionIntegration(t *testing.T) {
	reg, mf := initializeMetrics(t)

	conf := getty.GetDefaultClientConfig()
	conf.ConnectionNum = 1 // All attempts on a fixture share a real Session.
	conf.GrPoolSize = 0
	getty.SetClientConf(*conf)

	t.Run("success and request serialization failure", func(t *testing.T) {
		peer := newPeer(t, func(c net.Conn, req *remoting.Request) {
			writeResponse(t, c, req.ID, nil)
		})
		i := newInvoker(t, peer)
		reply := new(string)
		res := mf.Invoke(context.Background(), i, invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName("Success"), invocation.WithReply(reply)))
		if res.Error() != nil || *reply != "ok" {
			t.Fatalf("changed RPC result: %v / %q", res.Error(), *reply)
		}
		checkMetrics(t, reg, "Success", 1, 0, 0, 0)

		bad := mf.Invoke(context.Background(), i, invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName("EncodeFailure"), invocation.WithReply(new(string)),
			invocation.WithArguments([]any{make(chan int)})))
		if bad.Error() == nil || rpcerror.Of(bad.Error()) != rpcerror.Codec {
			t.Fatalf("real Hessian serialization failure lost origin: %v", bad.Error())
		}
		checkMetrics(t, reg, "EncodeFailure", 1, 1, 1, 0)
		// A following successful local attempt must not inherit Codec evidence.
		res = mf.Invoke(context.Background(), i, invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName("RetrySuccess"), invocation.WithReply(new(string))))
		if res.Error() != nil {
			t.Fatal(res.Error())
		}
		checkMetrics(t, reg, "RetrySuccess", 1, 0, 0, 0)
	})

	for _, trusted := range []bool{true, false} {
		name := "UntrustedHeader"
		if trusted {
			name = "DecodeFailure"
		}
		t.Run(name, func(t *testing.T) {
			peer := newPeer(t, func(c net.Conn, req *remoting.Request) {
				data := []byte{0xda, 0xbb, 2, 20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0xff}
				binary.BigEndian.PutUint64(data[4:], uint64(req.ID))
				if !trusted {
					data[0], data[1] = 0, 0
				}
				_, _ = c.Write(data)
			})
			i := newInvoker(t, peer)
			start := time.Now()
			res := mf.Invoke(context.Background(), i, invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName(name), invocation.WithReply(new(string))))
			if res.Error() == nil || time.Since(start) < 150*time.Millisecond {
				t.Fatalf("observation changed the original wait boundary: %v", res.Error())
			}
			// The user still sees the original wait error, not a replacement.
			if rpcerror.Of(res.Error()) != rpcerror.Unknown {
				t.Fatalf("sidecar evidence escaped into business error: %v", res.Error())
			}
			codec := float64(0)
			if trusted {
				codec = 1
			}
			checkMetrics(t, reg, name, 1, 1, codec, 0)
		})
	}

	t.Run("concurrent attempts share a session without sharing evidence", func(t *testing.T) {
		var requests []*remoting.Request
		goodDone := make(chan struct{})
		peer := newPeer(t, func(c net.Conn, req *remoting.Request) {
			requests = append(requests, req)
			if len(requests) != 2 {
				return
			}
			var badID int64
			for _, r := range requests {
				if r.Data.(base.Invocation).MethodName() == "ConcurrentGood" {
					writeResponse(t, c, r.ID, nil)
				} else {
					badID = r.ID
				}
			}
			<-goodDone
			data := []byte{0xda, 0xbb, 2, 20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0xff}
			binary.BigEndian.PutUint64(data[4:], uint64(badID))
			_, _ = c.Write(data)
		})
		i := newInvoker(t, peer)
		var wg sync.WaitGroup
		for _, method := range []string{"ConcurrentGood", "ConcurrentBad"} {
			wg.Go(func() {
				res := mf.Invoke(context.Background(), i, invocation.NewRPCInvocationWithOptions(
					invocation.WithMethodName(method), invocation.WithReply(new(string))))
				if method == "ConcurrentGood" {
					defer close(goodDone)
					if res.Error() != nil {
						t.Errorf("good attempt polluted: %v", res.Error())
					}
				} else if res.Error() == nil {
					t.Error("missing failed attempt")
				}
			})
		}
		wg.Wait()
		checkMetrics(t, reg, "ConcurrentGood", 1, 0, 0, 0)
		checkMetrics(t, reg, "ConcurrentBad", 1, 1, 1, 0)
	})

	t.Run("callback final response and expiry", func(t *testing.T) {
		release := make(chan struct{})
		defer close(release)
		requestID := make(chan int64, 1)
		peer := newPeer(t, func(c net.Conn, req *remoting.Request) {
			requestID <- req.ID
			<-release
			writeResponse(t, c, req.ID, nil)
		})
		i := newInvoker(t, peer)
		called := make(chan struct{}, 2)
		inv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName("LateCallback"), invocation.WithReply(new(string)),
			invocation.WithAttachment(constant.AsyncKey, "true"),
			invocation.WithCallBack(func(common.CallbackResponse) { called <- struct{}{} }))
		res := mf.Invoke(context.Background(), i, inv)
		if res.Error() != nil {
			t.Fatal(res.Error())
		}
		id := <-requestID
		await(t, func() bool { return metricValue(t, reg, "requests_processing_total", "LateCallback") == 1 })
		await(t, func() bool { return metricValue(t, reg, "requests_processing_total", "LateCallback") == 0 })
		checkMetrics(t, reg, "LateCallback", 0, 0, 0, 0)
		if remoting.GetPendingResponse(remoting.SequenceType(id)) == nil {
			t.Fatal("monitoring expiry removed the business callback")
		}
		release <- struct{}{}
		select {
		case <-called:
		case <-time.After(3 * time.Second):
			t.Fatal("late user callback suppressed")
		}
		if remoting.GetPendingResponse(remoting.SequenceType(id)) != nil {
			t.Fatal("real response did not retire the business pending entry")
		}
		checkMetrics(t, reg, "LateCallback", 0, 0, 0, 0)

		// A real failure response before expiry is counted once, after send.
		errorPeer := newPeer(t, func(c net.Conn, req *remoting.Request) {
			writeResponse(t, c, req.ID, errors.New("provider fixture failure"))
		})
		errorInvoker := newInvoker(t, errorPeer)
		errorInv := invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName("CallbackFailure"), invocation.WithReply(new(string)),
			invocation.WithAttachment(constant.AsyncKey, "true"),
			invocation.WithCallBack(func(common.CallbackResponse) { called <- struct{}{} }))
		if res := mf.Invoke(context.Background(), errorInvoker, errorInv); res.Error() != nil {
			t.Fatal("submission was changed into a response error", res.Error())
		}
		select {
		case <-called:
		case <-time.After(3 * time.Second):
			t.Fatal("missing user callback")
		}
		checkMetrics(t, reg, "CallbackFailure", 1, 1, 0, 0)
	})

	t.Run("real Triple dial failure", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		address := listener.Addr().String()
		_ = listener.Close()
		client := tri.NewClient(&http.Client{Timeout: time.Second}, "http://"+address+"/fixture.Service",
			tri.WithTriple(), tri.WithTimeout(time.Second))
		i := &functionInvoker{BaseInvoker: base.NewBaseInvoker(fixtureURL(t, address)),
			call: func(ctx context.Context, _ base.Invocation) result.Result {
				err := client.CallUnary(ctx, tri.NewRequest(wrapperspb.String("request")), "Call",
					tri.NewResponse(new(wrapperspb.StringValue)))
				return &result.RPCResult{Err: err}
			}}
		res := mf.Invoke(context.Background(), i, invocation.NewRPCInvocation("DialFailure", nil, nil))
		var op *net.OpError
		if res.Error() == nil || !errors.As(res.Error(), &op) || rpcerror.Of(res.Error()) != rpcerror.Network {
			t.Fatalf("real transport failure lost source/cause: %v", res.Error())
		}
		checkMetrics(t, reg, "DialFailure", 1, 1, 0, 1)
	})
}

func newInvoker(t *testing.T, address string) *dubbo.DubboInvoker {
	t.Helper()
	url := fixtureURL(t, address)
	client := getty.NewClient(getty.Options{ConnectTimeout: time.Second, RequestTimeout: 300 * time.Millisecond})
	exchange := remoting.NewExchangeClient(url, client, time.Second, false)
	if exchange == nil {
		t.Fatal("fixture connection failed")
	}
	t.Cleanup(exchange.Close)
	return dubbo.NewDubboInvoker(url, exchange)
}

// newPeer reads whole Dubbo frames, using the production request decoder.
func newPeer(t *testing.T, respond func(net.Conn, *remoting.Request)) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var conns []net.Conn
	stop := make(chan struct{})
	accepted := make(chan struct{})
	wg.Go(func() {
		defer close(accepted)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
			wg.Go(func() {
				defer conn.Close()
				for {
					header := make([]byte, 16)
					if _, err := io.ReadFull(conn, header); err != nil {
						return
					}
					body := make([]byte, binary.BigEndian.Uint32(header[12:]))
					if _, err := io.ReadFull(conn, body); err != nil {
						return
					}
					decoded, _, err := (&dubbo.DubboCodec{}).Decode(append(header, body...))
					if err != nil {
						t.Errorf("production peer decode: %v", err)
						return
					}
					select {
					case <-stop:
						return
					default:
					}
					respond(conn, decoded.Result.(*remoting.Request))
				}
			})
		}
	})
	t.Cleanup(func() {
		close(stop)
		_ = listener.Close()
		<-accepted
		mu.Lock()
		for _, c := range conns {
			_ = c.Close()
		}
		mu.Unlock()
		wg.Wait()
	})
	return listener.Addr().String()
}
func writeResponse(t *testing.T, c net.Conn, id int64, err error) {
	t.Helper()
	buf, encodeErr := (&dubbo.DubboCodec{}).EncodeResponse(&remoting.Response{
		ID: id, SerialID: constant.SHessian2, Status: 20,
		Result: &result.RPCResult{Rest: "ok", Err: err},
	})
	if encodeErr != nil {
		t.Errorf("production peer encode: %v", encodeErr)
		return
	}
	_, _ = c.Write(buf.Bytes())
}
