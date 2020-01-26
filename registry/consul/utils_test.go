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
	"github.com/hashicorp/consul/agent"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting"
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

type testConsulAgent struct {
	dataDir   string
	testAgent *agent.TestAgent
}

func newConsulAgent(t *testing.T, port int) *testConsulAgent {
	dataDir, _ := ioutil.TempDir("./", "agent")
	hcl := `
		ports { 
			http = ` + strconv.Itoa(port) + `
		}
		data_dir = "` + dataDir + `"
	`
	testAgent := &agent.TestAgent{Name: t.Name(), DataDir: dataDir, HCL: hcl}
	testAgent.Start(t)

	consulAgent := &testConsulAgent{
		dataDir:   dataDir,
		testAgent: testAgent,
	}
	return consulAgent
}

func (consulAgent *testConsulAgent) close() {
	consulAgent.testAgent.Shutdown()
	os.RemoveAll(consulAgent.dataDir)
}

type testServer struct {
	listener net.Listener
	wg       sync.WaitGroup
	done     chan struct{}
}

func newServer(host string, port int) *testServer {
	addr := fmt.Sprintf("%s:%d", host, port)
	tcpAddr, _ := net.ResolveTCPAddr("tcp", addr)
	listener, _ := net.ListenTCP("tcp", tcpAddr)

	server := &testServer{
		listener: listener,
		done:     make(chan struct{}),
	}

	server.wg.Add(1)
	go server.serve()
	return server
}

func (server *testServer) serve() {
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

func (server *testServer) close() {
	close(server.done)
	server.listener.Close()
	server.wg.Wait()
}

type consulRegistryTestSuite struct {
	t                *testing.T
	providerRegistry registry.Registry
	consumerRegistry *consulRegistry
	listener         registry.Listener
	providerUrl      common.URL
	consumerUrl      common.URL
}

func newConsulRegistryTestSuite(t *testing.T) *consulRegistryTestSuite {
	suite := &consulRegistryTestSuite{t: t}
	return suite
}

func (suite *consulRegistryTestSuite) close() {
	suite.listener.Close()
	suite.providerRegistry.Destroy()
	suite.consumerRegistry.Destroy()
}

// register -> subscribe -> unregister
func test1(t *testing.T) {
	consulAgent := newConsulAgent(t, registryPort)
	defer consulAgent.close()

	server := newServer(providerHost, providerPort)
	defer server.close()

	suite := newConsulRegistryTestSuite(t)
	defer suite.close()

	suite.testNewProviderRegistry()
	suite.testRegister()
	suite.testNewConsumerRegistry()
	suite.testSubscribe()
	suite.testListener(remoting.EventTypeAdd)
	suite.testUnregister()
	suite.testListener(remoting.EventTypeDel)
}

// subscribe -> register
func test2(t *testing.T) {
	consulAgent := newConsulAgent(t, registryPort)
	defer consulAgent.close()

	server := newServer(providerHost, providerPort)
	defer server.close()

	suite := newConsulRegistryTestSuite(t)
	defer suite.close()

	suite.testNewConsumerRegistry()
	suite.testSubscribe()
	suite.testNewProviderRegistry()
	suite.testRegister()
	suite.testListener(remoting.EventTypeAdd)
}

func TestConsulRegistry(t *testing.T) {
	t.Run("test1", test1)
	t.Run("test2", test2)
}
