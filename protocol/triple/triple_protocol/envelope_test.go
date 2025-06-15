// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triple_protocol

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

func TestEnvelopeRead(t *testing.T) {
	t.Parallel()

	prefixes := [5]byte{}
	payload := []byte(`{"number": 42}`)
	binary.BigEndian.PutUint32(prefixes[1:5], uint32(len(payload)))

	buf := &bytes.Buffer{}
	buf.Write(prefixes[:])
	buf.Write(payload)

	t.Run("full", func(t *testing.T) {
		t.Parallel()
		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{
			reader: bytes.NewReader(buf.Bytes()),
		}
		assert.Nil(t, rdr.Read(env))
		assert.Equal(t, payload, env.Data.Bytes())
	})
	t.Run("byteByByte", func(t *testing.T) {
		t.Parallel()
		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{
			reader: byteByByteReader{
				reader: bytes.NewReader(buf.Bytes()),
			},
		}
		assert.Nil(t, rdr.Read(env))
		assert.Equal(t, payload, env.Data.Bytes())
	})
}

// byteByByteReader is test reader that reads a single byte at a time.
type byteByByteReader struct {
	reader io.ByteReader
}

func (b byteByByteReader) Read(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	next, err := b.reader.ReadByte()
	if err != nil {
		return 0, err
	}
	data[0] = next
	return 1, nil
}
