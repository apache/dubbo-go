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

package dubbo

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func TestResponseDecodeFailureCorrelation(t *testing.T) {
	id := remoting.SequenceID()
	_, o := rpcerror.WithObservation(context.Background())
	session, other := new(int), new(int)
	o.BindSession(session)
	p := remoting.NewPendingResponse(id)
	p.Observation = o
	p.Reply = new(string)
	remoting.AddPendingResponse(p)
	defer remoting.RemovePendingResponse(remoting.SequenceType(id))
	data := []byte{0xda, 0xbb, 2, 20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0xff}
	binary.BigEndian.PutUint64(data[4:], uint64(id))
	_, _, err := (&DubboCodec{}).Decode(data)
	var decode *remoting.ResponseDecodeError
	if !errors.As(err, &decode) || decode.RequestID != id || rpcerror.Of(err) != rpcerror.Codec {
		t.Fatalf("missing correlation: %v", err)
	}
	if remoting.ObserveResponseDecodeFailure(id, other, err) || !remoting.ObserveResponseDecodeFailure(id, session, err) {
		t.Fatal("incorrect session attribution")
	}
	select {
	case <-p.Done:
		t.Fatal("observation completed business wait")
	default:
	}
	if o.Freeze() != rpcerror.Codec {
		t.Fatal("missing frozen origin")
	}
	// Error-status responses still have a trustworthy ID and codec boundary.
	data[3] = 70
	_, _, err = (&DubboCodec{}).Decode(data)
	if !errors.As(err, &decode) || decode.RequestID != id || rpcerror.Of(err) != rpcerror.Codec {
		t.Fatalf("error-status body lost correlation: %v", err)
	}
	data[3] = 20
	// Partial frame and invalid magic must not provide trusted correlation.
	for _, bad := range [][]byte{data[:10], append([]byte(nil), data...)} {
		if len(bad) == len(data) {
			bad[0] = 0
		}
		_, _, err = (&DubboCodec{}).Decode(bad)
		if errors.As(err, &decode) {
			t.Fatal("untrusted header attributed to a request")
		}
	}
}
