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

package codecplugin_test

import (
	"strconv"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

func TestServiceUnexportDrainsCodecAndKeepsSibling(t *testing.T) {
	defer extension.GetProtocol("tri").Destroy()
	for _, wire := range []string{"grpc", "triple"} {
		t.Run(wire, func(t *testing.T) {
			port := freePort(t)
			prefix := "retire-" + wire + strconv.Itoa(port)
			groupA, groupB := prefix+"-a", prefix+"-b"
			tagA, tagB := prefix+"-A:", prefix+"-B:"
			entered, gate := make(chan struct{}), make(chan struct{})
			released := false
			defer func() {
				if !released {
					close(gate)
				}
			}()
			a := handlerContextServer(t, port, groupA, "", tagA, &HandlerContextEcho{entered: entered, gate: gate})
			ready(t, port)
			handlerContextServer(t, port, groupB, "", tagB, &HandlerContextEcho{})
			ca := handlerContextClient(t, port, groupA, "", tagA, wire)
			cb := handlerContextClient(t, port, groupB, "", tagB, wire)
			type outcome struct {
				reply string
				err   error
			}
			result := make(chan outcome, 1)
			go func() { reply, err := handlerContextCall(ca, "blocked"); result <- outcome{reply, err} }()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("request did not enter")
			}
			done := make(chan struct{})
			go func() { a.Unexport(); close(done) }()
			retired := false
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				if _, err := handlerContextCall(ca, "new"); err != nil {
					retired = true
					break
				}
				time.Sleep(time.Millisecond)
			}
			if !retired {
				t.Fatal("retired service still accepts new calls")
			}
			select {
			case <-done:
				t.Fatal("Unexport did not wait for the active request")
			default:
			}
			if count("close/"+tagA+"/provider").Load() != 0 {
				t.Fatal("closed active codec")
			}
			if reply, err := handlerContextCall(cb, "sibling"); err != nil || reply != "hello sibling" {
				t.Fatalf("sibling reply=%s err=%v", reply, err)
			}
			close(gate)
			released = true
			select {
			case got := <-result:
				if got.err != nil || got.reply != "hello blocked" {
					t.Fatalf("inflight reply=%s err=%v", got.reply, got.err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("inflight request did not finish")
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("Unexport did not finish")
			}
			if count("close/"+tagA+"/provider").Load() != 1 {
				t.Fatal("Unexport did not release its plugin exactly once")
			}
			tagNew := prefix + "-new:"
			next := handlerContextServer(t, port, groupA, "", tagNew, &HandlerContextEcho{})
			cn := handlerContextClient(t, port, groupA, "", tagNew, wire)
			a.Unexport()
			if reply, err := handlerContextCall(cn, "replacement"); err != nil || reply != "hello replacement" {
				t.Fatalf("replacement reply=%s err=%v", reply, err)
			}
			next.Unexport()
			next.Unexport()
			if count("close/"+tagNew+"/provider").Load() != 1 {
				t.Fatal("repeated Unexport closed plugin more than once")
			}
			if count("close/"+tagB+"/provider").Load() != 0 {
				t.Fatal("sibling codec closed")
			}
		})
	}
}
