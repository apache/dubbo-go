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

package impl

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type failingSerializer struct{ err error }

func (s failingSerializer) Marshal(DubboPackage) ([]byte, error)  { return nil, s.err }
func (s failingSerializer) Unmarshal([]byte, *DubboPackage) error { return s.err }

func TestSerializationFailureOrigin(t *testing.T) {
	cause := errors.New("serialization fixture")
	s := failingSerializer{cause}
	for _, encode := range []func(DubboPackage, Serializer) ([]byte, error){packRequest, packResponse} {
		_, err := encode(DubboPackage{Header: DubboHeader{Type: PackageRequest_TwoWay}}, s)
		if !errors.Is(err, cause) || rpcerror.Of(err) != rpcerror.Codec || err.Error() != cause.Error() {
			t.Fatalf("lost serialization cause: %v", err)
		}
	}
}

func TestLateResponseWithoutPending(t *testing.T) {
	id := remoting.SequenceID()
	data := append([]byte(nil), DubboResponseHeaderBytes[:]...)
	data[2] |= 2 // Hessian2 serialization ID.
	binary.BigEndian.PutUint64(data[4:], uint64(id))
	binary.BigEndian.PutUint32(data[12:], 1)
	data = append(data, 0xff) // Do not decode a retired caller's body/Reply.
	pkg := NewDubboPackage(bytes.NewBuffer(data))
	if err := pkg.Unmarshal(); err != nil {
		t.Fatal(err)
	}
	if _, ok := pkg.Body.(*ResponsePayload); !ok {
		t.Fatal("late frame must have a safe empty response")
	}
}
