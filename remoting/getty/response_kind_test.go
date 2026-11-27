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

package getty

import (
	"context"
	"errors"
	"testing"
	"time"
)

import (
	dubboGetty "github.com/apache/dubbo-getty"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// respondingSession answers every written request with the configured response.
type respondingSession struct {
	dubboGetty.Session
	respond func(id int64)
}

func (s *respondingSession) WritePkg(pkg any, _ time.Duration) (int, int, error) {
	go s.respond(pkg.(*remoting.Request).ID)
	return 0, 0, nil
}

func (s *respondingSession) IsClosed() bool { return false }

func (s *respondingSession) Stat() string { return "responding-session" }

func (s *respondingSession) Close() {}

// A synchronous request returns the decoded error unchanged and records the
// response's declared kind on the observation it actually returned.
func TestRequestContextRecordsResponseKind(t *testing.T) {
	raw := errors.New("insufficient balance")
	client := NewClient(Options{RequestTimeout: time.Second})
	client.gettyClient = &gettyRPCClient{
		rpcClient: client,
		sessions: []*rpcSession{{session: &respondingSession{respond: func(id int64) {
			(&remoting.Response{ID: id, Error: raw, Result: &result.RPCResult{Err: raw}, Kind: rpcerror.KindBusiness}).Handle()
		}}}},
	}

	_, o := rpcerror.WithObservation(context.Background())
	got := make(chan rpcerror.Completion, 1)
	o.OnComplete(func(c rpcerror.Completion) { got <- c })

	request := remoting.NewRequest("2.0.2")
	request.TwoWay = true
	rsp := remoting.NewPendingResponse(request.ID)
	rsp.Observation = o
	remoting.AddPendingResponse(rsp)

	err := client.RequestContext(context.Background(), request, time.Second, rsp)
	require.ErrorIs(t, err, raw)
	assert.Equal(t, rpcerror.KindUnknown, rpcerror.KindOf(err), "the returned error must not be wrapped")

	o.FinishInvoke(err, true)
	select {
	case c := <-got:
		assert.Equal(t, rpcerror.KindBusiness, c.Kind)
	case <-time.After(time.Second):
		t.Fatal("observation did not complete")
	}
}
