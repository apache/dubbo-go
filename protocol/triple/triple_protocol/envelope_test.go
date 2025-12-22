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
	"errors"
	"io"
	"strings"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

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

// mockEnvelopeCodec implements Codec for testing
type mockEnvelopeCodec struct {
	name         string
	marshalErr   error
	unmarshalErr error
	marshalData  []byte
}

func (m *mockEnvelopeCodec) Name() string {
	return m.name
}

func (m *mockEnvelopeCodec) Marshal(v any) ([]byte, error) {
	if m.marshalErr != nil {
		return nil, m.marshalErr
	}
	if m.marshalData != nil {
		return m.marshalData, nil
	}
	return []byte("marshaled"), nil
}

func (m *mockEnvelopeCodec) Unmarshal(data []byte, v any) error {
	return m.unmarshalErr
}

// mockEnvelopeWriter implements io.Writer for testing
type mockEnvelopeWriter struct {
	buf           *bytes.Buffer
	writeErr      error
	writeErrAfter int
	bytesWritten  int
}

func (m *mockEnvelopeWriter) Write(p []byte) (n int, err error) {
	if m.writeErr != nil && m.writeErrAfter == 0 {
		return 0, m.writeErr
	}
	if m.writeErrAfter > 0 {
		m.bytesWritten += len(p)
		if m.bytesWritten > m.writeErrAfter {
			return 0, errors.New("write error after threshold")
		}
	}
	return m.buf.Write(p)
}

// errorReader always returns an error
type errorReader struct {
	err error
}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

// limitedReader reads up to a limit then returns EOF
type limitedReader struct {
	data  []byte
	pos   int
	limit int
}

func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.pos >= len(l.data) || l.pos >= l.limit {
		return 0, io.EOF
	}
	n = copy(p, l.data[l.pos:])
	if l.pos+n > l.limit {
		n = l.limit - l.pos
	}
	l.pos += n
	return n, nil
}

// TestEnvelopeIsSet tests envelope.IsSet method
func TestEnvelopeIsSet(t *testing.T) {
	t.Parallel()

	t.Run("FlagSet", func(t *testing.T) {
		t.Parallel()
		env := &envelope{Flags: flagEnvelopeCompressed}
		assert.True(t, env.IsSet(flagEnvelopeCompressed))
	})

	t.Run("FlagNotSet", func(t *testing.T) {
		t.Parallel()
		env := &envelope{Flags: 0}
		assert.False(t, env.IsSet(flagEnvelopeCompressed))
	})

	t.Run("MultipleFlags", func(t *testing.T) {
		t.Parallel()
		env := &envelope{Flags: 0b00000011}
		assert.True(t, env.IsSet(flagEnvelopeCompressed))
		assert.True(t, env.IsSet(0b00000010))
	})
}

// TestEnvelopeWriterMarshal tests envelopeWriter.Marshal method
func TestEnvelopeWriterMarshal(t *testing.T) {
	t.Parallel()

	t.Run("NilMessage", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		w := &envelopeWriter{
			writer:     buf,
			codec:      &mockEnvelopeCodec{name: "mock"},
			bufferPool: newBufferPool(),
		}

		err := w.Marshal(nil)
		assert.Nil(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		w := &envelopeWriter{
			writer:     buf,
			codec:      &mockEnvelopeCodec{name: "mock", marshalData: []byte("test")},
			bufferPool: newBufferPool(),
		}

		err := w.Marshal("test message")
		assert.Nil(t, err)
		assert.True(t, buf.Len() > 0)
	})

	t.Run("MarshalError", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		w := &envelopeWriter{
			writer:     buf,
			codec:      &mockEnvelopeCodec{name: "mock", marshalErr: errors.New("marshal error")},
			bufferPool: newBufferPool(),
		}

		err := w.Marshal("test message")
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeInternal)
	})

	t.Run("MarshalErrorWithBackupCodec", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		w := &envelopeWriter{
			writer:      buf,
			codec:       &mockEnvelopeCodec{name: "primary", marshalErr: errors.New("marshal error")},
			backupCodec: &mockEnvelopeCodec{name: "backup", marshalData: []byte("backup")},
			bufferPool:  newBufferPool(),
		}

		err := w.Marshal("test message")
		assert.Nil(t, err)
	})

	t.Run("WriteError", func(t *testing.T) {
		t.Parallel()
		w := &envelopeWriter{
			writer:     &mockEnvelopeWriter{buf: &bytes.Buffer{}, writeErr: errors.New("write error")},
			codec:      &mockEnvelopeCodec{name: "mock", marshalData: []byte("test")},
			bufferPool: newBufferPool(),
		}

		err := w.Marshal("test message")
		assert.NotNil(t, err)
	})
}

// TestEnvelopeWriterWrite tests envelopeWriter.Write method
func TestEnvelopeWriterWrite(t *testing.T) {
	t.Parallel()

	t.Run("UncompressedSmallMessage", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		w := &envelopeWriter{
			writer:           buf,
			codec:            &mockEnvelopeCodec{name: "mock"},
			bufferPool:       newBufferPool(),
			compressMinBytes: 100,
		}

		env := &envelope{Data: bytes.NewBuffer([]byte("small"))}
		err := w.Write(env)
		assert.Nil(t, err)
		assert.True(t, buf.Len() > 0)
	})

	t.Run("AlreadyCompressed", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		w := &envelopeWriter{
			writer:           buf,
			codec:            &mockEnvelopeCodec{name: "mock"},
			bufferPool:       newBufferPool(),
			compressMinBytes: 1,
		}

		env := &envelope{
			Data:  bytes.NewBuffer([]byte("data")),
			Flags: flagEnvelopeCompressed,
		}
		err := w.Write(env)
		assert.Nil(t, err)
	})

	t.Run("ExceedsSendMaxBytes", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		w := &envelopeWriter{
			writer:       buf,
			codec:        &mockEnvelopeCodec{name: "mock"},
			bufferPool:   newBufferPool(),
			sendMaxBytes: 5,
		}

		env := &envelope{Data: bytes.NewBuffer([]byte("this is a long message"))}
		err := w.Write(env)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeResourceExhausted)
	})

	t.Run("NoCompressionPool", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		w := &envelopeWriter{
			writer:          buf,
			codec:           &mockEnvelopeCodec{name: "mock"},
			bufferPool:      newBufferPool(),
			compressionPool: nil,
		}

		env := &envelope{Data: bytes.NewBuffer([]byte("data"))}
		err := w.Write(env)
		assert.Nil(t, err)
	})
}

// TestEnvelopeWriterWriteError tests write with writer errors
func TestEnvelopeWriterWriteError(t *testing.T) {
	t.Parallel()

	t.Run("PrefixWriteError", func(t *testing.T) {
		t.Parallel()
		w := &envelopeWriter{
			writer:     &mockEnvelopeWriter{buf: &bytes.Buffer{}, writeErr: errors.New("write error")},
			bufferPool: newBufferPool(),
		}

		env := &envelope{Data: bytes.NewBuffer([]byte("data"))}
		err := w.write(env)
		assert.NotNil(t, err)
	})
}

// TestEnvelopeReaderUnmarshal tests envelopeReader.Unmarshal method
func TestEnvelopeReaderUnmarshal(t *testing.T) {
	t.Parallel()

	t.Run("EmptyMessage", func(t *testing.T) {
		t.Parallel()
		prefixes := [5]byte{}
		buf := bytes.NewBuffer(prefixes[:])

		r := &envelopeReader{
			reader:     buf,
			codec:      &mockEnvelopeCodec{name: "mock"},
			bufferPool: newBufferPool(),
		}

		var msg string
		err := r.Unmarshal(&msg)
		assert.Nil(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		payload := []byte("test data")
		prefixes := [5]byte{}
		binary.BigEndian.PutUint32(prefixes[1:5], uint32(len(payload)))

		buf := &bytes.Buffer{}
		buf.Write(prefixes[:])
		buf.Write(payload)

		r := &envelopeReader{
			reader:     buf,
			codec:      &mockEnvelopeCodec{name: "mock"},
			bufferPool: newBufferPool(),
		}

		var msg string
		err := r.Unmarshal(&msg)
		assert.Nil(t, err)
	})

	t.Run("UnmarshalError", func(t *testing.T) {
		t.Parallel()
		payload := []byte("test data")
		prefixes := [5]byte{}
		binary.BigEndian.PutUint32(prefixes[1:5], uint32(len(payload)))

		buf := &bytes.Buffer{}
		buf.Write(prefixes[:])
		buf.Write(payload)

		r := &envelopeReader{
			reader:     buf,
			codec:      &mockEnvelopeCodec{name: "mock", unmarshalErr: errors.New("unmarshal error")},
			bufferPool: newBufferPool(),
		}

		var msg string
		err := r.Unmarshal(&msg)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeInvalidArgument)
	})

	t.Run("UnmarshalErrorWithBackupCodec", func(t *testing.T) {
		t.Parallel()
		payload := []byte("test data")
		prefixes := [5]byte{}
		binary.BigEndian.PutUint32(prefixes[1:5], uint32(len(payload)))

		buf := &bytes.Buffer{}
		buf.Write(prefixes[:])
		buf.Write(payload)

		r := &envelopeReader{
			reader:      buf,
			codec:       &mockEnvelopeCodec{name: "primary", unmarshalErr: errors.New("unmarshal error")},
			backupCodec: &mockEnvelopeCodec{name: "backup"},
			bufferPool:  newBufferPool(),
		}

		var msg string
		err := r.Unmarshal(&msg)
		assert.Nil(t, err)
	})

	t.Run("CompressedWithoutPool", func(t *testing.T) {
		t.Parallel()
		payload := []byte("compressed data")
		prefixes := [5]byte{flagEnvelopeCompressed}
		binary.BigEndian.PutUint32(prefixes[1:5], uint32(len(payload)))

		buf := &bytes.Buffer{}
		buf.Write(prefixes[:])
		buf.Write(payload)

		r := &envelopeReader{
			reader:          buf,
			codec:           &mockEnvelopeCodec{name: "mock"},
			bufferPool:      newBufferPool(),
			compressionPool: nil,
		}

		var msg string
		err := r.Unmarshal(&msg)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeInvalidArgument)
	})

	t.Run("SpecialEnvelope", func(t *testing.T) {
		t.Parallel()
		payload := []byte("special data")
		prefixes := [5]byte{0b10000000}
		binary.BigEndian.PutUint32(prefixes[1:5], uint32(len(payload)))

		buf := &bytes.Buffer{}
		buf.Write(prefixes[:])
		buf.Write(payload)

		r := &envelopeReader{
			reader:     buf,
			codec:      &mockEnvelopeCodec{name: "mock"},
			bufferPool: newBufferPool(),
		}

		var msg string
		err := r.Unmarshal(&msg)
		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, io.EOF))
	})
}

// TestEnvelopeRead tests envelopeReader.Read method
func TestEnvelopeRead(t *testing.T) {
	t.Parallel()

	prefixes := [5]byte{}
	payload := []byte(`{"number": 42}`)
	binary.BigEndian.PutUint32(prefixes[1:5], uint32(len(payload)))

	buf := &bytes.Buffer{}
	buf.Write(prefixes[:])
	buf.Write(payload)

	t.Run("Full", func(t *testing.T) {
		t.Parallel()
		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{reader: bytes.NewReader(buf.Bytes())}
		assert.Nil(t, rdr.Read(env))
		assert.Equal(t, payload, env.Data.Bytes())
	})

	t.Run("ByteByByte", func(t *testing.T) {
		t.Parallel()
		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{
			reader: byteByByteReader{reader: bytes.NewReader(buf.Bytes())},
		}
		assert.Nil(t, rdr.Read(env))
		assert.Equal(t, payload, env.Data.Bytes())
	})

	t.Run("ZeroSizePrefix", func(t *testing.T) {
		t.Parallel()
		prefixes := [5]byte{}
		buf := bytes.NewBuffer(prefixes[:])

		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{reader: buf}
		err := rdr.Read(env)
		assert.Nil(t, err)
		assert.Equal(t, env.Data.Len(), 0)
	})

	t.Run("EmptyReader", func(t *testing.T) {
		t.Parallel()
		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{reader: bytes.NewReader([]byte{})}
		err := rdr.Read(env)
		assert.NotNil(t, err)
	})

	t.Run("IncompletePrefix", func(t *testing.T) {
		t.Parallel()
		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{reader: bytes.NewReader([]byte{0, 0, 0})}
		err := rdr.Read(env)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeInvalidArgument)
	})

	t.Run("ExceedsReadMaxBytes", func(t *testing.T) {
		t.Parallel()
		prefixes := [5]byte{}
		binary.BigEndian.PutUint32(prefixes[1:5], 1000)

		buf := &bytes.Buffer{}
		buf.Write(prefixes[:])
		buf.Write(make([]byte, 1000))

		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{reader: buf, readMaxBytes: 100}
		err := rdr.Read(env)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeResourceExhausted)
	})

	t.Run("IncompletePayload", func(t *testing.T) {
		t.Parallel()
		prefixes := [5]byte{}
		binary.BigEndian.PutUint32(prefixes[1:5], 100)

		buf := &bytes.Buffer{}
		buf.Write(prefixes[:])
		buf.Write([]byte("short"))

		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{reader: buf}
		err := rdr.Read(env)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeInvalidArgument)
	})

	t.Run("WithFlags", func(t *testing.T) {
		t.Parallel()
		payload := []byte("data")
		prefixes := [5]byte{flagEnvelopeCompressed}
		binary.BigEndian.PutUint32(prefixes[1:5], uint32(len(payload)))

		buf := &bytes.Buffer{}
		buf.Write(prefixes[:])
		buf.Write(payload)

		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{reader: buf}
		err := rdr.Read(env)
		assert.Nil(t, err)
		assert.Equal(t, env.Flags, uint8(flagEnvelopeCompressed))
	})

	t.Run("ReadError", func(t *testing.T) {
		t.Parallel()
		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{reader: errorReader{err: errors.New("read error")}}
		err := rdr.Read(env)
		assert.NotNil(t, err)
	})

	t.Run("PartialPayload", func(t *testing.T) {
		t.Parallel()
		prefixes := [5]byte{}
		binary.BigEndian.PutUint32(prefixes[1:5], 100)

		data := make([]byte, 105)
		copy(data, prefixes[:])
		copy(data[5:], strings.Repeat("x", 50))

		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{reader: &limitedReader{data: data, limit: 55}}
		err := rdr.Read(env)
		assert.NotNil(t, err)
	})
}

// TestIsSizeZeroPrefix tests isSizeZeroPrefix function
func TestIsSizeZeroPrefix(t *testing.T) {
	t.Parallel()

	t.Run("AllZeros", func(t *testing.T) {
		t.Parallel()
		prefix := [5]byte{0, 0, 0, 0, 0}
		assert.True(t, isSizeZeroPrefix(prefix))
	})

	t.Run("FlagSetSizeZero", func(t *testing.T) {
		t.Parallel()
		prefix := [5]byte{flagEnvelopeCompressed, 0, 0, 0, 0}
		assert.True(t, isSizeZeroPrefix(prefix))
	})

	t.Run("NonZeroSize", func(t *testing.T) {
		t.Parallel()
		prefix := [5]byte{0, 0, 0, 0, 1}
		assert.False(t, isSizeZeroPrefix(prefix))
	})

	t.Run("LargeSize", func(t *testing.T) {
		t.Parallel()
		prefix := [5]byte{}
		binary.BigEndian.PutUint32(prefix[1:5], 1000)
		assert.False(t, isSizeZeroPrefix(prefix))
	})
}

// TestEnvelopeConstants tests envelope constants
func TestEnvelopeConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int(flagEnvelopeCompressed), 0b00000001)
}

// TestEnvelopeWriterWriteWithCompression tests Write with compression enabled
func TestEnvelopeWriterWriteWithCompression(t *testing.T) {
	t.Parallel()

	t.Run("CompressionError", func(t *testing.T) {
		t.Parallel()
		buf := &bytes.Buffer{}
		pool := newCompressionPool(
			func() Decompressor { return nil },
			func() Compressor {
				return &envTestCompressorWithError{compressErr: errors.New("compress error")}
			},
		)
		w := &envelopeWriter{
			writer:           buf,
			codec:            &mockEnvelopeCodec{name: "mock"},
			bufferPool:       newBufferPool(),
			compressionPool:  pool,
			compressMinBytes: 1,
		}

		largeData := bytes.Repeat([]byte("x"), 100)
		env := &envelope{Data: bytes.NewBuffer(largeData)}
		err := w.Write(env)
		assert.NotNil(t, err)
	})
}

// envTestCompressorWithError implements Compressor that returns error
type envTestCompressorWithError struct {
	compressErr error
	target      io.Writer
}

func (m *envTestCompressorWithError) Write(p []byte) (n int, err error) {
	if m.compressErr != nil {
		return 0, m.compressErr
	}
	return m.target.Write(p)
}

func (m *envTestCompressorWithError) Close() error {
	return nil
}

func (m *envTestCompressorWithError) Reset(w io.Writer) {
	m.target = w
}

// TestEnvelopeWriterWriteDataCopyError tests write with data copy error
func TestEnvelopeWriterWriteDataCopyError(t *testing.T) {
	t.Parallel()

	t.Run("DataCopyError", func(t *testing.T) {
		t.Parallel()
		w := &envelopeWriter{
			writer:     &mockEnvelopeWriter{buf: &bytes.Buffer{}, writeErr: nil, writeErrAfter: 5},
			bufferPool: newBufferPool(),
		}

		env := &envelope{Data: bytes.NewBuffer([]byte("data"))}
		err := w.write(env)
		assert.NotNil(t, err)
	})
}

// TestEnvelopeReaderUnmarshalWithDecompression tests Unmarshal with decompression
func TestEnvelopeReaderUnmarshalWithDecompression(t *testing.T) {
	t.Parallel()

	t.Run("DecompressionError", func(t *testing.T) {
		t.Parallel()
		payload := []byte("compressed data")
		prefixes := [5]byte{flagEnvelopeCompressed}
		binary.BigEndian.PutUint32(prefixes[1:5], uint32(len(payload)))

		buf := &bytes.Buffer{}
		buf.Write(prefixes[:])
		buf.Write(payload)

		pool := newCompressionPool(
			func() Decompressor {
				return &envTestDecompressor{decompressErr: errors.New("decompress error")}
			},
			func() Compressor { return nil },
		)

		r := &envelopeReader{
			reader:          buf,
			codec:           &mockEnvelopeCodec{name: "mock"},
			bufferPool:      newBufferPool(),
			compressionPool: pool,
		}

		var msg string
		err := r.Unmarshal(&msg)
		assert.NotNil(t, err)
	})

	t.Run("SuccessfulDecompression", func(t *testing.T) {
		t.Parallel()
		payload := []byte("compressed data")
		prefixes := [5]byte{flagEnvelopeCompressed}
		binary.BigEndian.PutUint32(prefixes[1:5], uint32(len(payload)))

		buf := &bytes.Buffer{}
		buf.Write(prefixes[:])
		buf.Write(payload)

		pool := newCompressionPool(
			func() Decompressor {
				return &envTestDecompressor{data: []byte("decompressed")}
			},
			func() Compressor { return nil },
		)

		r := &envelopeReader{
			reader:          buf,
			codec:           &mockEnvelopeCodec{name: "mock"},
			bufferPool:      newBufferPool(),
			compressionPool: pool,
		}

		var msg string
		err := r.Unmarshal(&msg)
		assert.Nil(t, err)
	})
}

// envTestDecompressor implements Decompressor for testing
type envTestDecompressor struct {
	data          []byte
	decompressErr error
	pos           int
}

func (m *envTestDecompressor) Read(p []byte) (n int, err error) {
	if m.decompressErr != nil {
		return 0, m.decompressErr
	}
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *envTestDecompressor) Close() error {
	return nil
}

func (m *envTestDecompressor) Reset(r io.Reader) error {
	m.pos = 0
	return nil
}

// TestEnvelopeReaderReadMaxBytesError tests Read with maxBytesError
func TestEnvelopeReaderReadMaxBytesError(t *testing.T) {
	t.Parallel()

	t.Run("DiscardErrorOnExceedsMax", func(t *testing.T) {
		t.Parallel()
		prefixes := [5]byte{}
		binary.BigEndian.PutUint32(prefixes[1:5], 1000)

		buf := &bytes.Buffer{}
		buf.Write(prefixes[:])

		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{
			reader:       &discardErrorReader{data: buf.Bytes(), pos: 0},
			readMaxBytes: 100,
		}
		err := rdr.Read(env)
		assert.NotNil(t, err)
	})
}

// discardErrorReader returns error when discarding
type discardErrorReader struct {
	data []byte
	pos  int
}

func (d *discardErrorReader) Read(p []byte) (n int, err error) {
	if d.pos < 5 {
		n = copy(p, d.data[d.pos:])
		d.pos += n
		return n, nil
	}
	return 0, errors.New("discard error")
}

// TestEnvelopeWriterMarshalNilWriteError tests Marshal with nil message write error
func TestEnvelopeWriterMarshalNilWriteError(t *testing.T) {
	t.Parallel()

	t.Run("NilMessageWriteTripleError", func(t *testing.T) {
		t.Parallel()
		w := &envelopeWriter{
			writer:     &tripleErrorWriter{},
			codec:      &mockEnvelopeCodec{name: "mock"},
			bufferPool: newBufferPool(),
		}

		err := w.Marshal(nil)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeInternal)
	})
}

// tripleErrorWriter returns a triple Error on write
type tripleErrorWriter struct{}

func (t *tripleErrorWriter) Write(p []byte) (n int, err error) {
	return 0, NewError(CodeInternal, errors.New("triple error"))
}

// TestEnvelopeReaderReadTripleError tests Read with triple error
func TestEnvelopeReaderReadTripleError(t *testing.T) {
	t.Parallel()

	t.Run("TripleErrorOnRead", func(t *testing.T) {
		t.Parallel()
		env := &envelope{Data: &bytes.Buffer{}}
		rdr := envelopeReader{reader: &tripleErrorReader{}}
		err := rdr.Read(env)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeInternal)
	})
}

// tripleErrorReader returns a triple Error on read
type tripleErrorReader struct{}

func (t *tripleErrorReader) Read(p []byte) (n int, err error) {
	return 0, NewError(CodeInternal, errors.New("triple error"))
}

// TestEnvelopeWriterWriteTripleError tests write with triple error
func TestEnvelopeWriterWriteTripleError(t *testing.T) {
	t.Parallel()

	t.Run("TripleErrorOnPrefixWrite", func(t *testing.T) {
		t.Parallel()
		w := &envelopeWriter{
			writer:     &tripleErrorWriter{},
			bufferPool: newBufferPool(),
		}

		env := &envelope{Data: bytes.NewBuffer([]byte("data"))}
		err := w.write(env)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeInternal)
	})
}

// Update mockEnvelopeWriter to support writeErrAfter
func init() {
	// This is just to ensure the test file compiles
}

// TestEnvelopeReaderUnmarshalCopyError tests Unmarshal with copy error for special envelope
func TestEnvelopeReaderUnmarshalCopyError(t *testing.T) {
	t.Parallel()

	t.Run("SpecialEnvelopeWithData", func(t *testing.T) {
		t.Parallel()
		payload := []byte("special data")
		prefixes := [5]byte{0b10000000}
		binary.BigEndian.PutUint32(prefixes[1:5], uint32(len(payload)))

		buf := &bytes.Buffer{}
		buf.Write(prefixes[:])
		buf.Write(payload)

		r := &envelopeReader{
			reader:     buf,
			codec:      &mockEnvelopeCodec{name: "mock"},
			bufferPool: newBufferPool(),
		}

		var msg string
		err := r.Unmarshal(&msg)
		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, io.EOF))
	})
}
