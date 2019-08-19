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
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/registry"
)

import (
	"github.com/hashicorp/consul/agent"
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
		done:     make(chan struct{}),
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

type ConsulRegistryTestSuite struct {
	t                *testing.T
	dataDir          string
	consulAgent      *agent.TestAgent
	providerRegistry registry.Registry
	consumerRegistry registry.Registry
	listener         registry.Listener
	providerUrl      common.URL
	server           *Server
}

func newConsulRegistryTestSuite(t *testing.T) *ConsulRegistryTestSuite {
	dataDir, _ := ioutil.TempDir("./", "agent")
	hcl := `
		ports { 
			http = ` + strconv.Itoa(registryPort) + `
		}
		data_dir = "` + dataDir + `"
	`
	consulAgent := &agent.TestAgent{Name: t.Name(), DataDir: dataDir, HCL: hcl}
	consulAgent.Start(t)

	suite := &ConsulRegistryTestSuite{
		t:           t,
		dataDir:     dataDir,
		consulAgent: consulAgent,
	}
	return suite
}

func (suite *ConsulRegistryTestSuite) close() {
	suite.server.close()
	suite.consulAgent.Shutdown()
	os.RemoveAll(suite.dataDir)
}

func TestConsulRegistry(t *testing.T) {
	suite := newConsulRegistryTestSuite(t)
	defer suite.close()
	suite.testNewProviderRegistry()
	suite.testSubscribe()
	suite.testNewConsumerRegistry()
	suite.testRegister()
	suite.testListener()
}
