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

package serialization

import (
	"errors"
	"io"
	"sync"
)

// Resources owns codecs constructed for one binding. Owners drain their RPCs
// before Close. Failed initialization closes already-created codecs in reverse.
type Resources struct {
	once    sync.Once
	mu      sync.Mutex
	active  sync.WaitGroup
	closing bool
	closers []io.Closer
	err     error
}

func (r *Resources) Add(c Codec) {
	if closer, ok := c.(io.Closer); ok {
		r.closers = append(r.closers, closer)
	}
}
func (r *Resources) Close() error {
	if r == nil {
		return nil
	}
	r.once.Do(func() {
		r.mu.Lock()
		r.closing = true
		r.mu.Unlock()
		r.active.Wait()
		for i := len(r.closers) - 1; i >= 0; i-- {
			r.err = errors.Join(r.err, r.closers[i].Close())
		}
	})
	return r.err
}

func (r *Resources) Begin() bool {
	if r == nil {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closing {
		return false
	}
	r.active.Add(1)
	return true
}
func (r *Resources) End() {
	if r != nil {
		r.active.Done()
	}
}
func (r *Resources) HasClosers() bool { return r != nil && len(r.closers) > 0 }
