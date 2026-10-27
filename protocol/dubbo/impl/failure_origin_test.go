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
	"errors"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
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
