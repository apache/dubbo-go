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

package customizer

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func TestHostPortCustomizerGetPriority(t *testing.T) {
	c := &hostPortCustomizer{}
	assert.Equal(t, 1, c.GetPriority())
}

// TestHostPortCustomizerPortAlreadySet verifies that an instance with a port
// already set is not modified.
func TestHostPortCustomizerPortAlreadySet(t *testing.T) {
	c := &hostPortCustomizer{}
	ins := &registry.DefaultServiceInstance{
		Port:            20880,
		ServiceMetadata: newTestMetadataInfo(newHostPortTestURL("dubbo", "20880", "127.0.0.1")),
	}
	c.Customize(ins)
	assert.Equal(t, 20880, ins.Port)
	assert.Empty(t, ins.Host, "host should not be modified when port is already set")
}

// TestHostPortCustomizerNilServiceMetadata verifies that nothing happens when
// the instance carries no service metadata.
func TestHostPortCustomizerNilServiceMetadata(t *testing.T) {
	c := &hostPortCustomizer{}
	ins := &registry.DefaultServiceInstance{}
	c.Customize(ins)
	assert.Equal(t, 0, ins.Port)
	assert.Empty(t, ins.Host)
}

// TestHostPortCustomizerNoExportedURLs verifies that nothing happens when the
// metadata info has no exported service URLs.
func TestHostPortCustomizerNoExportedURLs(t *testing.T) {
	c := &hostPortCustomizer{}
	ins := &registry.DefaultServiceInstance{
		ServiceMetadata: info.NewMetadataInfo("app", ""),
	}
	c.Customize(ins)
	assert.Equal(t, 0, ins.Port)
	assert.Empty(t, ins.Host)
}

// TestHostPortCustomizerNormal verifies that the host and port are taken from
// the first exported service URL.
func TestHostPortCustomizerNormal(t *testing.T) {
	c := &hostPortCustomizer{}
	ins := &registry.DefaultServiceInstance{
		ServiceMetadata: newTestMetadataInfo(newHostPortTestURL("dubbo", "20880", "127.0.0.1")),
	}
	c.Customize(ins)
	assert.Equal(t, "127.0.0.1", ins.Host)
	assert.Equal(t, 20880, ins.Port)
}

// TestHostPortCustomizerOnlyFirstURL verifies that only the first exported URL
// is used.
func TestHostPortCustomizerOnlyFirstURL(t *testing.T) {
	c := &hostPortCustomizer{}
	mi := info.NewMetadataInfo("app", "")
	mi.AddService(newHostPortTestURL("dubbo", "20880", "127.0.0.1"))
	mi.AddService(newHostPortTestURL("tri", "50051", "10.0.0.1"))
	ins := &registry.DefaultServiceInstance{ServiceMetadata: mi}
	c.Customize(ins)
	assert.Equal(t, "127.0.0.1", ins.Host)
	assert.Equal(t, 20880, ins.Port)
}

// TestHostPortCustomizerUnparsablePort verifies that an unparsable port leaves
// the port unchanged while the host is still set.
func TestHostPortCustomizerUnparsablePort(t *testing.T) {
	c := &hostPortCustomizer{}
	ins := &registry.DefaultServiceInstance{
		ServiceMetadata: newTestMetadataInfo(newHostPortTestURL("dubbo", "not-a-number", "127.0.0.1")),
	}
	c.Customize(ins)
	assert.Equal(t, "127.0.0.1", ins.Host)
	assert.Equal(t, 0, ins.Port, "unparsable port should leave the port unchanged")
}

// TestHostPortCustomizerNonDefaultInstance verifies that a non-DefaultServiceInstance
// is never modified, even when it carries exported URLs.
func TestHostPortCustomizerNonDefaultInstance(t *testing.T) {
	c := &hostPortCustomizer{}
	ins := &wrappedServiceInstance{
		DefaultServiceInstance: &registry.DefaultServiceInstance{
			ServiceMetadata: newTestMetadataInfo(newHostPortTestURL("dubbo", "20880", "127.0.0.1")),
		},
	}
	c.Customize(ins)
	assert.Equal(t, 0, ins.Port)
	assert.Empty(t, ins.Host)
}

// wrappedServiceInstance wraps DefaultServiceInstance so that the type assertion
// to *registry.DefaultServiceInstance fails while still implementing the interface.
type wrappedServiceInstance struct {
	*registry.DefaultServiceInstance
}

func newTestMetadataInfo(url *common.URL) *info.MetadataInfo {
	mi := info.NewMetadataInfo("app", "")
	mi.AddService(url)
	return mi
}

func newHostPortTestURL(protocol, port, ip string) *common.URL {
	return common.NewURLWithOptions(
		common.WithProtocol(protocol),
		common.WithPort(port),
		common.WithIp(ip),
	)
}
