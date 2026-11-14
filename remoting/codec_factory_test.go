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
	"bytes"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type factoryTestCodec struct{ tag string }

func (*factoryTestCodec) EncodeRequest(*Request) (*bytes.Buffer, error) {
	return bytes.NewBufferString("request"), nil
}
func (*factoryTestCodec) EncodeResponse(*Response) (*bytes.Buffer, error) {
	return bytes.NewBufferString("response"), nil
}
func (*factoryTestCodec) Decode([]byte) (*DecodeResult, int, error) { return nil, 0, nil }
func TestCodecFactoryAndLegacyOverride(t *testing.T) {
	const protocol = "unit-factory"
	url := common.NewURLWithOptions(common.WithProtocol(protocol), common.WithParamsValue("tag", "original"))
	var constructed *factoryTestCodec
	reg := CodecRegistration{Protocol: protocol, Name: protocol, New: func(snapshot *common.URL) (Codec, error) {
		constructed = &factoryTestCodec{tag: snapshot.GetParam("tag", "")}
		snapshot.SetParam("tag", "mutated")
		return constructed, nil
	}}
	require.NoError(t, RegisterCodec(reg))
	require.ErrorContains(t, RegisterCodec(reg), "already registered")
	require.NoError(t, PrepareCodec(url))
	url.SetParam("tag", "after")
	bound, err := NewCodec(url)
	require.NoError(t, err)
	require.Same(t, constructed, bound)
	require.Equal(t, "original", constructed.tag)
	legacy := &factoryTestCodec{tag: "legacy"}
	RegistryCodec(protocol, legacy)
	require.Same(t, legacy, GetCodec(protocol))
	fresh := common.NewURLWithOptions(common.WithProtocol(protocol))
	bound, err = NewCodec(fresh)
	require.NoError(t, err)
	require.Same(t, legacy, bound)
	// Existing prepared views are unaffected by subsequent legacy registration.
	bound, err = NewCodec(url)
	require.NoError(t, err)
	require.Same(t, constructed, bound)
	fresh.SetParam("codec", "missing")
	_, err = NewCodec(fresh)
	require.ErrorContains(t, err, "unknown codec")
	_, err = NewCodec(nil)
	require.ErrorContains(t, err, "nil codec URL")
}

type closeFactoryCodec struct {
	factoryTestCodec
	closes int
}

func (c *closeFactoryCodec) Close() error { c.closes++; return nil }

func TestSharedListenerCodecOwnership(t *testing.T) {
	listener := common.NewURLWithOptions(common.WithProtocol("shared-listener"))
	first := &closeFactoryCodec{}
	listener.SetAttribute(CodecAttribute, &codecBinding{codec: first})
	snapshot := listener.Clone()
	require.NoError(t, PrepareCodec(listener))
	original, _ := snapshot.GetAttribute(CodecAttribute)
	prepared, _ := listener.GetAttribute(CodecAttribute)
	require.Same(t, original, prepared, "preparation must preserve ownership identity")
	candidate := &closeFactoryCodec{}
	service := common.NewURLWithOptions(common.WithProtocol("shared-listener"))
	service.SetAttribute(CodecAttribute, &codecBinding{codec: candidate})
	require.NoError(t, ReuseCodec(service, snapshot))
	require.Equal(t, 1, candidate.closes, "dispose of a racing unused candidate")
	require.NoError(t, ReleaseCodec(service))
	require.Zero(t, first.closes, "failed service export must not close listener's codec")
	listener.ClearAttributes() // exporter destruction clears the original URL
	require.NoError(t, ReleaseCodec(snapshot))
	require.NoError(t, ReleaseCodec(snapshot))
	require.Equal(t, 1, first.closes, "listener snapshot owns exactly one close")
}

func TestClientCodecLeasesAndLocalRecreation(t *testing.T) {
	const protocol = "unit-client-lease"
	var instances []*closeFactoryCodec
	require.NoError(t, RegisterCodec(CodecRegistration{Protocol: protocol, Name: protocol, New: func(url *common.URL) (Codec, error) {
		require.Equal(t, "LOCAL", url.GetParam("tag", ""))
		c := &closeFactoryCodec{}
		instances = append(instances, c)
		return c, nil
	}}))
	ref := common.NewURLWithOptions(common.WithProtocol(protocol), common.WithParamsValue("tag", "LOCAL"))
	require.NoError(t, PrepareCodec(ref))
	provider := common.NewURLWithOptions(common.WithProtocol(protocol), common.WithParamsValue("tag", "REMOTE")).MergeURL(ref)
	first, releaseFirst, err := AcquireCodec(provider)
	require.NoError(t, err)
	second, releaseSecond, err := AcquireCodec(provider.Clone())
	require.NoError(t, err)
	require.Same(t, first, second)
	require.NoError(t, releaseFirst())
	require.NoError(t, releaseFirst())
	require.Zero(t, instances[0].closes)
	require.NoError(t, ReleaseCodec(ref))
	require.Zero(t, instances[0].closes)
	require.NoError(t, releaseSecond())
	require.Equal(t, 1, instances[0].closes)
	next, releaseNext, err := AcquireCodec(provider.Clone())
	require.NoError(t, err)
	require.NotSame(t, first, next)
	require.Len(t, instances, 2)
	require.NoError(t, releaseNext())
	require.Equal(t, 1, instances[1].closes)
}

func TestClientCodecConcurrentRelease(t *testing.T) {
	codec := &closeFactoryCodec{}
	ref := common.NewURLWithOptions(common.WithProtocol("unit-lease-concurrent"))
	ref.SetAttribute(CodecAttribute, &codecBinding{codec: codec})
	const clients = 24
	releases := make([]func() error, clients)
	for i := range releases {
		_, release, err := AcquireCodec(ref.Clone())
		require.NoError(t, err)
		releases[i] = release
	}
	var wg sync.WaitGroup
	for _, release := range releases {
		wg.Add(1)
		go func(release func() error) { defer wg.Done(); _ = release(); _ = release() }(release)
	}
	wg.Wait()
	require.Equal(t, 1, codec.closes)
}

type codecGenerationProbe struct {
	factoryTestCodec
	entered, gate chan struct{}
	closes        atomic.Int64
}

func (c *codecGenerationProbe) Close() error {
	c.closes.Add(1)
	if c.entered != nil {
		close(c.entered)
		<-c.gate
	}
	return nil
}
func TestCodecGenerationInterleaveAndRetry(t *testing.T) {
	const name = "codec-generation-retry"
	sentinel := errors.New("rebuild failed")
	old := &codecGenerationProbe{entered: make(chan struct{}), gate: make(chan struct{})}
	next := &codecGenerationProbe{}
	failed := &codecGenerationProbe{}
	var unblock sync.Once
	defer unblock.Do(func() { close(old.gate) })
	count := 0
	require.NoError(t, RegisterCodec(CodecRegistration{Protocol: name, Name: name, New: func(url *common.URL) (Codec, error) {
		require.Equal(t, "local", url.GetParam("tag", ""))
		count++
		switch count {
		case 1:
			return old, nil
		case 2:
			return failed, sentinel
		default:
			return next, nil
		}
	}}))
	ref := common.NewURLWithOptions(common.WithProtocol(name), common.WithParamsValue("tag", "local"))
	require.NoError(t, PrepareCodec(ref))
	remote := common.NewURLWithOptions(common.WithProtocol(name), common.WithParamsValue("tag", "remote")).MergeURL(ref)
	_, release, err := AcquireCodec(remote)
	require.NoError(t, err)
	done := make(chan struct{})
	go func() { _ = release(); close(done) }()
	select {
	case <-old.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("old Close not entered")
	}
	_, _, err = AcquireCodec(remote.Clone())
	require.ErrorIs(t, err, sentinel)
	require.EqualValues(t, 1, failed.closes.Load(), "failed factory result must be closed")
	codec, releaseNext, err := AcquireCodec(remote.Clone())
	require.NoError(t, err)
	require.Same(t, next, codec)
	_, err = codec.EncodeRequest(nil)
	require.NoError(t, err)
	require.Zero(t, next.closes.Load())
	unblock.Do(func() { close(old.gate) })
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("old generation did not close")
	}
	require.NoError(t, release())
	require.Zero(t, next.closes.Load())
	require.NoError(t, releaseNext())
	require.EqualValues(t, 1, next.closes.Load())
}

func TestCodecRemotingFactoryResults(t *testing.T) {
	sentinel := errors.New("remoting construction sentinel")
	for _, test := range []struct {
		name   string
		result func(*closeFactoryCodec) (Codec, error)
		bad    bool
	}{
		{"nil", func(*closeFactoryCodec) (Codec, error) { return nil, nil }, true},
		{"typed-nil", func(*closeFactoryCodec) (Codec, error) { return (*closeFactoryCodec)(nil), nil }, true},
		{"error", func(*closeFactoryCodec) (Codec, error) { return nil, sentinel }, true},
		{"instance-error", func(c *closeFactoryCodec) (Codec, error) { return c, sentinel }, true},
		{"success", func(c *closeFactoryCodec) (Codec, error) { return c, nil }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			name := "codec-remoting-result-" + test.name
			c := &closeFactoryCodec{}
			var returned Codec
			var factoryErr error
			require.NoError(t, RegisterCodec(CodecRegistration{Protocol: name, Name: name, New: func(*common.URL) (Codec, error) { returned, factoryErr = test.result(c); return returned, factoryErr }}))
			got, err := NewCodec(common.NewURLWithOptions(common.WithProtocol(name)))
			if test.bad {
				require.Error(t, err)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Same(t, c, got)
			}
			if factoryErr != nil {
				require.ErrorIs(t, err, sentinel)
			}
			expected := 0
			if test.bad && !codecIsNil(returned) {
				expected = 1
			}
			require.Equal(t, expected, c.closes)
		})
	}
}
