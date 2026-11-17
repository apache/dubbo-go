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

package triple_protocol

import (
	"testing"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/encoding/protowire"

	"google.golang.org/protobuf/proto"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/interoperability"
)

func TestConfiguredResponseWrapperCompatibility(t *testing.T) {
	inner := runtimeJSON{}
	payload, err := inner.Marshal("reply")
	require.NoError(t, err)
	codec := &protoWrapperCodec{innerCodec: inner, runtimeCodecs: map[string]Codec{inner.Name(): inner}}
	for _, withType := range []bool{false, true} {
		normal := &interoperability.TripleResponseWrapper{SerializeType: inner.Name(), Data: payload}
		legacy := &interoperability.TripleRequestWrapper{SerializeType: inner.Name(), Args: [][]byte{payload}}
		if withType {
			normal.Type = "java.lang.String"
			legacy.ArgTypes = []string{"java.lang.String"}
		}
		for _, message := range []proto.Message{normal, legacy} {
			data, marshalErr := proto.Marshal(message)
			require.NoError(t, marshalErr)
			var reply string
			require.NoError(t, codec.Unmarshal(data, &reply))
			require.Equal(t, "reply", reply)
			denied := &protoWrapperCodec{innerCodec: inner, runtimeCodecs: map[string]Codec{}}
			require.EqualError(t, denied.Unmarshal(data, &reply), `wrapper codec: unsupported or disabled serialize type "private-json"`)
		}
	}
	for _, message := range []proto.Message{
		&interoperability.TripleResponseWrapper{SerializeType: inner.Name()},
		&interoperability.TripleRequestWrapper{SerializeType: inner.Name(), ArgTypes: []string{"void"}},
	} {
		data, marshalErr := proto.Marshal(message)
		require.NoError(t, marshalErr)
		require.NoError(t, codec.Unmarshal(data, new(string)))
	}
}

func TestConfiguredResponseRejectsRepeatedPayloads(t *testing.T) {
	probe := &codecDecodeProbe{name: "probe"}
	codec := &protoWrapperCodec{innerCodec: probe, runtimeCodecs: map[string]Codec{"probe": probe}}
	for _, payloads := range [][][]byte{{[]byte("a"), []byte("b")}, {nil, []byte("b")}, {nil, nil}} {
		data, err := proto.Marshal(&interoperability.TripleResponseWrapper{SerializeType: "probe"})
		require.NoError(t, err)
		for _, payload := range payloads {
			data = protowire.AppendTag(data, 2, protowire.BytesType)
			data = protowire.AppendBytes(data, payload)
		}
		require.ErrorContains(t, codec.Unmarshal(data, new(string)), "multiple values")
		require.Zero(t, probe.calls)
	}
	require.ErrorContains(t, codec.Unmarshal([]byte{0xff}, new(string)), "invalid response wrapper")
}
