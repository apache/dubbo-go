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
)

import (
	getty "github.com/apache/dubbo-getty"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// Only identity is needed: this unit test does not start Getty's TCP lifecycle.
type decodeTestSession struct{ getty.Session }

type responseErrorCodec struct {
	remoting.Codec
	err error
}

func (c responseErrorCodec) Decode([]byte) (*remoting.DecodeResult, int, error) {
	return nil, 17, c.err
}

func TestClientReadCorrelatesOnlyValidatedResponseErrors(t *testing.T) {
	cause := rpcerror.Mark(errors.New("codec fixture"), rpcerror.Codec)
	for _, tc := range []struct {
		name    string
		trusted bool
		same    bool
		want    rpcerror.Origin
	}{
		{"validated response on sending session", true, true, rpcerror.Codec},
		{"validated response on another session", true, false, rpcerror.Unknown},
		{"uncorrelated codec error", false, true, rpcerror.Unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := remoting.SequenceID()
			_, o := rpcerror.WithObservation(context.Background())
			session := &decodeTestSession{}
			o.BindSession(session)
			pending := remoting.NewPendingResponse(id)
			pending.Observation = o
			remoting.AddPendingResponse(pending)
			t.Cleanup(func() { remoting.RemovePendingResponse(remoting.SequenceType(id)) })
			err := cause
			if tc.trusted {
				err = &remoting.ResponseDecodeError{RequestID: id, Cause: cause}
			}
			readSession := session
			if !tc.same {
				readSession = &decodeTestSession{}
			}
			handler := NewRpcClientPackageHandler(&Client{codec: responseErrorCodec{err: err}})
			decoded, n, got := handler.Read(readSession, nil)
			if decoded != nil || n != 17 || !errors.Is(got, err) {
				t.Fatalf("Read changed decoder result: %v / %d / %v", decoded, n, got)
			}
			if o.Freeze() != tc.want || remoting.GetPendingResponse(remoting.SequenceType(id)) != pending {
				t.Fatal("read attributed to the wrong session or retired the pending request")
			}
			select {
			case <-pending.Done:
				t.Fatal("read completed the business waiter")
			default:
			}
		})
	}
}
