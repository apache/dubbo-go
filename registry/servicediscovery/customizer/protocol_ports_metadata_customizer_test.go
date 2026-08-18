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
	"encoding/json"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func TestProtocolPortsMetadataCustomizerGetPriority(t *testing.T) {
	p := &ProtocolPortsMetadataCustomizer{}
	assert.Equal(t, 0, p.GetPriority())
}

// TestProtocolPortsCustomizeEmptyList verifies that no endpoints are written
// when there are no exported service URLs (client side).
func TestProtocolPortsCustomizeEmptyList(t *testing.T) {
	p := &ProtocolPortsMetadataCustomizer{}
	ins := &registry.DefaultServiceInstance{}
	p.Customize(ins)
	_, ok := ins.GetMetadata()[constant.ServiceInstanceEndpoints]
	assert.False(t, ok, "endpoints should not be written for an empty exported URL list")
}

// TestProtocolPortsCustomizeWithURLs verifies that endpoints are derived from
// the exported service URLs and written as a JSON array.
func TestProtocolPortsCustomizeWithURLs(t *testing.T) {
	urlDubbo := newEndpointTestURL("dubbo", "20880")
	urlTri := newEndpointTestURL("tri", "50051")
	metadata.AddService("protocol-ports-test", urlDubbo)
	metadata.AddService("protocol-ports-test", urlTri)
	t.Cleanup(func() {
		metadata.RemoveService("protocol-ports-test", urlDubbo)
		metadata.RemoveService("protocol-ports-test", urlTri)
	})

	p := &ProtocolPortsMetadataCustomizer{}
	ins := &registry.DefaultServiceInstance{}
	p.Customize(ins)

	str := ins.GetMetadata()[constant.ServiceInstanceEndpoints]
	assert.NotEmpty(t, str)
	var endpoints []registry.Endpoint
	assert.NoError(t, json.Unmarshal([]byte(str), &endpoints))
	assert.Len(t, endpoints, 2)

	got := make(map[string]int)
	for _, e := range endpoints {
		got[e.Protocol] = e.Port
	}
	assert.Equal(t, 20880, got["dubbo"])
	assert.Equal(t, 50051, got["tri"])
}

// TestProtocolPortsCustomizeSkipsEmptyProtocol verifies that URLs with an empty
// protocol are skipped and do not appear in the endpoints.
func TestProtocolPortsCustomizeSkipsEmptyProtocol(t *testing.T) {
	urlNoProtocol := newEndpointTestURL("", "20880")
	urlDubbo := newEndpointTestURL("dubbo", "20881")
	metadata.AddService("protocol-ports-skip", urlNoProtocol)
	metadata.AddService("protocol-ports-skip", urlDubbo)
	t.Cleanup(func() {
		metadata.RemoveService("protocol-ports-skip", urlNoProtocol)
		metadata.RemoveService("protocol-ports-skip", urlDubbo)
	})

	p := &ProtocolPortsMetadataCustomizer{}
	ins := &registry.DefaultServiceInstance{}
	p.Customize(ins)

	str := ins.GetMetadata()[constant.ServiceInstanceEndpoints]
	var endpoints []registry.Endpoint
	assert.NoError(t, json.Unmarshal([]byte(str), &endpoints))
	assert.Len(t, endpoints, 1, "URL with empty protocol should be skipped")
	assert.Equal(t, "dubbo", endpoints[0].Protocol)
	assert.Equal(t, 20881, endpoints[0].Port)
}

// TestProtocolPortsCustomizeUnparsablePort verifies that an unparsable port is
// recorded as 0 (the endpoint is still kept) and does not abort the whole write.
func TestProtocolPortsCustomizeUnparsablePort(t *testing.T) {
	urlBadPort := newEndpointTestURL("dubbo", "not-a-number")
	urlTri := newEndpointTestURL("tri", "50051")
	metadata.AddService("protocol-ports-badport", urlBadPort)
	metadata.AddService("protocol-ports-badport", urlTri)
	t.Cleanup(func() {
		metadata.RemoveService("protocol-ports-badport", urlBadPort)
		metadata.RemoveService("protocol-ports-badport", urlTri)
	})

	p := &ProtocolPortsMetadataCustomizer{}
	ins := &registry.DefaultServiceInstance{}
	p.Customize(ins)

	str := ins.GetMetadata()[constant.ServiceInstanceEndpoints]
	var endpoints []registry.Endpoint
	assert.NoError(t, json.Unmarshal([]byte(str), &endpoints))
	assert.Len(t, endpoints, 2)

	ports := make(map[string]int)
	for _, e := range endpoints {
		ports[e.Protocol] = e.Port
	}
	assert.Equal(t, 0, ports["dubbo"], "unparsable port should be recorded as 0")
	assert.Equal(t, 50051, ports["tri"], "other endpoints should still be written")
}

func TestEndpointsStrEmpty(t *testing.T) {
	assert.Equal(t, "", endpointsStr(map[string]int{}))
	assert.Equal(t, "", endpointsStr(nil))
}

func TestEndpointsStrNormal(t *testing.T) {
	str := endpointsStr(map[string]int{"dubbo": 123})
	var endpoints []registry.Endpoint
	assert.NoError(t, json.Unmarshal([]byte(str), &endpoints))
	assert.Len(t, endpoints, 1)
	assert.Equal(t, "dubbo", endpoints[0].Protocol)
	assert.Equal(t, 123, endpoints[0].Port)
}

// newEndpointTestURL builds a URL with the given protocol and port for testing.
func newEndpointTestURL(protocol, port string) *common.URL {
	return common.NewURLWithOptions(
		common.WithInterface("org.example.TestService"),
		common.WithProtocol(protocol),
		common.WithPort(port),
	)
}
