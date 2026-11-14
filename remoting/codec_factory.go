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

package remoting

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type CodecFactory func(*common.URL) (Codec, error)
type CodecRegistration struct {
	Protocol, Name string
	New            CodecFactory
}

var codecFactories = struct {
	sync.RWMutex
	items map[string]CodecRegistration
}{items: make(map[string]CodecRegistration)}
var codecNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

func RegisterCodec(reg CodecRegistration) error {
	if !codecNamePattern.MatchString(reg.Protocol) || !codecNamePattern.MatchString(reg.Name) || reg.New == nil {
		return fmt.Errorf("remoting: invalid codec registration %q/%q", reg.Protocol, reg.Name)
	}
	codecFactories.Lock()
	defer codecFactories.Unlock()
	key := reg.Protocol + "/" + reg.Name
	if _, ok := codecFactories.items[key]; ok {
		return fmt.Errorf("remoting: codec %s already registered", key)
	}
	codecFactories.items[key] = reg
	return nil
}
func MustRegisterCodec(reg CodecRegistration) {
	if err := RegisterCodec(reg); err != nil {
		panic(err)
	}
}

const CodecAttribute = "remoting.codec.binding"

type codecBinding struct {
	mu      sync.Mutex
	codec   Codec
	source  *common.URL
	users   int
	retired bool
	owner   *codecOwner
}

type codecOwner struct {
	codec     Codec
	closeOnce sync.Once
	closeErr  error
}

const borrowedCodecAttribute = "remoting.codec.borrowed"

// NewCodec constructs a local binding from a URL snapshot. Legacy default
// singleton registration remains available when no named factory is installed.
func NewCodec(url *common.URL) (Codec, error) {
	if url == nil {
		return nil, fmt.Errorf("remoting: nil codec URL")
	}
	if raw, ok := url.GetAttribute(CodecAttribute); ok {
		if binding, ok := raw.(*codecBinding); ok {
			binding.mu.Lock()
			defer binding.mu.Unlock()
			return binding.codec, nil
		}
	}
	name := url.GetParam("codec", url.Protocol)
	codecMu.RLock()
	legacy, overridden := legacyCodecOverrides[url.Protocol]
	codecMu.RUnlock()
	if name == url.Protocol && overridden {
		if codecIsNil(legacy) {
			return nil, fmt.Errorf("remoting: legacy codec is nil")
		}
		return legacy, nil
	}
	codecFactories.RLock()
	reg, ok := codecFactories.items[url.Protocol+"/"+name]
	codecFactories.RUnlock()
	var c Codec
	var err error
	if ok {
		c, err = reg.New(url.Clone())
	} else if name == url.Protocol {
		c = GetCodec(url.Protocol)
	} else {
		return nil, fmt.Errorf("remoting: unknown codec %s/%s", url.Protocol, name)
	}
	if err != nil {
		if !codecIsNil(c) {
			if closer, ok := c.(io.Closer); ok {
				_ = closer.Close()
			}
		}
		return nil, fmt.Errorf("remoting codec %s: %w", name, err)
	}
	if codecIsNil(c) {
		return nil, fmt.Errorf("remoting codec %s: factory returned nil", name)
	}
	return c, nil
}
func PrepareCodec(url *common.URL) error {
	if url != nil {
		if raw, ok := url.GetAttribute(CodecAttribute); ok {
			if _, valid := raw.(*codecBinding); valid {
				return nil
			}
		}
	}
	c, err := NewCodec(url)
	if err != nil {
		return err
	}
	url.SetAttribute(CodecAttribute, &codecBinding{codec: c, source: url.Clone(), owner: &codecOwner{codec: c}})
	return nil
}

func codecIsNil(c Codec) bool {
	if c == nil {
		return true
	}
	v := reflect.ValueOf(c)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// ReuseCodec borrows an existing listener binding. If a racing export already
// constructed an unused candidate, release that candidate before replacing it.
func ReuseCodec(url, listener *common.URL) error {
	raw, ok := listener.GetAttribute(CodecAttribute)
	binding, valid := raw.(*codecBinding)
	if !ok || !valid {
		return fmt.Errorf("remoting: listener has no prepared codec")
	}
	if previous, ok := url.GetAttribute(CodecAttribute); ok && previous != raw {
		if err := ReleaseCodec(url); err != nil {
			return err
		}
	}
	url.SetAttribute(CodecAttribute, binding)
	if url != listener {
		url.SetAttribute(borrowedCodecAttribute, true)
	}
	return nil
}

// ReleaseCodec rolls back a prepared owner. Borrowed service URLs never own the
// endpoint binding; only its listener releases it after sessions have stopped.
func ReleaseCodec(url *common.URL) error {
	if url == nil {
		return nil
	}
	if borrowed, _ := url.GetAttribute(borrowedCodecAttribute); borrowed == true {
		return nil
	}
	if raw, ok := url.GetAttribute(CodecAttribute); ok {
		if binding, ok := raw.(*codecBinding); ok {
			binding.mu.Lock()
			if binding.users > 0 {
				binding.mu.Unlock()
				return nil
			}
			binding.retired = true
			owner := binding.codecOwner()
			binding.mu.Unlock()
			return owner.close()
		}
	}
	return nil
}

// codecOwner is called while holding the binding lock. Each generation has its
// own close guard, so final release can drain independently of rediscovery.
func (b *codecBinding) codecOwner() *codecOwner {
	if b.owner == nil {
		b.owner = &codecOwner{codec: b.codec}
	}
	return b.owner
}
func (o *codecOwner) close() error {
	o.closeOnce.Do(func() {
		if closer, ok := o.codec.(io.Closer); ok {
			o.closeErr = closer.Close()
		}
	})
	return o.closeErr
}

// AcquireCodec leases a reference-scoped codec to one transport client. Provider
// URL copies share the binding, while release is idempotent for each client.
func AcquireCodec(url *common.URL) (Codec, func() error, error) {
	if err := PrepareCodec(url); err != nil {
		return nil, nil, err
	}
	raw, _ := url.GetAttribute(CodecAttribute)
	binding := raw.(*codecBinding)
	binding.mu.Lock()
	if binding.retired {
		if binding.source == nil {
			binding.mu.Unlock()
			return nil, nil, fmt.Errorf("remoting: released codec has no construction snapshot")
		}
		codec, err := NewCodec(binding.source.Clone())
		if err != nil {
			binding.mu.Unlock()
			return nil, nil, err
		}
		binding.codec = codec
		binding.owner = &codecOwner{codec: codec}
		binding.retired = false
	}
	binding.users++
	owner := binding.codecOwner()
	binding.mu.Unlock()
	var once sync.Once
	var releaseErr error
	release := func() error {
		once.Do(func() {
			binding.mu.Lock()
			binding.users--
			last := binding.users == 0
			if last {
				binding.retired = true
			}
			binding.mu.Unlock()
			if last {
				releaseErr = owner.close()
			}
		})
		return releaseErr
	}
	return owner.codec, release, nil
}
