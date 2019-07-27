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

package consul

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"
	"testing"
)

import (
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

var (
	registryHost = "localhost"
	registryPort = 8500
	providerHost = "localhost"
	providerPort = 8000
	consumerHost = "localhost"
	consumerPort = 8001
	service      = "HelloWorld"
	protocol     = "tcp"
)

func newProviderRegistryUrl(host string, port int) *common.URL {
	url1 := common.NewURLWithOptions(
		common.WithIp(host),
		common.WithPort(strconv.Itoa(port)),
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)),
	)
	return url1
}

func newConsumerRegistryUrl(host string, port int) *common.URL {
	url1 := common.NewURLWithOptions(
		common.WithIp(host),
		common.WithPort(strconv.Itoa(port)),
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.CONSUMER)),
	)
	return url1
}

func newProviderUrl(host string, port int, service string, protocol string) common.URL {
	url1 := common.NewURLWithOptions(
		common.WithIp(host),
		common.WithPort(strconv.Itoa(port)),
		common.WithPath(service),
		common.WithProtocol(protocol),
	)
	return *url1
}

func newConsumerUrl(host string, port int, service string, protocol string) common.URL {
	url1 := common.NewURLWithOptions(
		common.WithIp(host),
		common.WithPort(strconv.Itoa(port)),
		common.WithPath(service),
		common.WithProtocol(protocol),
	)
	return *url1
}

type Server struct {
	listener net.Listener
	wg       sync.WaitGroup
	done     chan struct{}
}

func newServer(host string, port int) *Server {
	addr := fmt.Sprintf("%s:%d", host, port)
	tcpAddr, _ := net.ResolveTCPAddr("tcp", addr)
	listener, _ := net.ListenTCP("tcp", tcpAddr)

	server := &Server{
		listener: listener,
		done:     make(chan struct{}, 1),
	}

	server.wg.Add(1)
	go server.serve()
	return server
}

func (server *Server) serve() {
	defer server.wg.Done()
	for {
		select {
		case <-server.done:
			return
		default:
			conn, err := server.listener.Accept()
			if err != nil {
				continue
			}
			conn.Write([]byte("Hello World"))
			conn.Close()
		}
	}
}

func (server *Server) close() {
	close(server.done)
	server.listener.Close()
	server.wg.Wait()
}

func TestSomething(t *testing.T) {
	providerRegistryUrl := newProviderRegistryUrl(registryHost, registryPort)
	consumerRegistryUrl := newConsumerRegistryUrl(registryHost, registryPort)
	providerUrl := newProviderUrl(providerHost, providerPort, service, protocol)
	consumerUrl := newConsumerUrl(consumerHost, consumerPort, service, protocol)

	cb := func(c *testutil.TestServerConfig) { c.Ports.HTTP = registryPort }
	consulServer, _ := testutil.NewTestServerConfig(cb)
	defer consulServer.Stop()
	providerRegistry, err := newConsulRegistry(providerRegistryUrl)
	assert.NoError(t, err)
	consumerRegistry, err := newConsulRegistry(consumerRegistryUrl)
	assert.NoError(t, err)

	server := newServer(providerHost, providerPort)
	defer server.close()
	err = providerRegistry.Register(providerUrl)
	assert.NoError(t, err)

	listener, err := consumerRegistry.Subscribe(consumerUrl)
	assert.NoError(t, err)
	event, err := listener.Next()
	assert.NoError(t, err)
	assert.True(t, providerUrl.URLEqual(event.Service))
}
