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
	"bytes"
	"sync"
)

const (
	initialBufferSize    = 512
	maxRecycleBufferSize = 8 * 1024 * 1024 // Don't recycle buffers larger than this.
)

type bufferPool struct {
	sync.Pool
}

func newBufferPool() *bufferPool {
	return &bufferPool{
		Pool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, initialBufferSize))
			},
		},
	}
}

func (b *bufferPool) Get() *bytes.Buffer {
	if buf, ok := b.Pool.Get().(*bytes.Buffer); ok {
		return buf
	}
	return bytes.NewBuffer(make([]byte, 0, initialBufferSize))
}

func (b *bufferPool) Put(buffer *bytes.Buffer) {
	if buffer.Cap() > maxRecycleBufferSize {
		return
	}
	buffer.Reset()
	b.Pool.Put(buffer)
}

// marshalToPool serializes message with appender into a *bytes.Buffer drawn
// from pool and returns it. If the pooled array is too small and the appender
// grows the slice, the larger array is swapped in so it can be recycled once
// the caller returns the buffer. On failure the buffer is put back and a
// CodeInternal error is returned.
func marshalToPool(pool *bufferPool, appender marshalAppender, message any) (*bytes.Buffer, *Error) {
	buffer := pool.Get()
	raw, err := appender.MarshalAppend(buffer.Bytes(), message)
	if err != nil {
		pool.Put(buffer)
		return nil, errorf(CodeInternal, "marshal message: %w", err)
	}
	if cap(raw) > buffer.Cap() {
		// MarshalAppend grew the slice: swap the larger array in so it can be
		// recycled next time.
		*buffer = *bytes.NewBuffer(raw)
	} else {
		// No reallocation occurred; adopt the bytes the appender already wrote
		// into the pooled array.
		buffer.Write(raw)
	}
	return buffer, nil
}
