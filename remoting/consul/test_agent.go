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
	"io/ioutil"
	"os"
	"strconv"
	"testing"
)

import (
	"github.com/hashicorp/consul/agent"
)

// Consul agent, used for test, simulates
// an embedded consul server.
type ConsulAgent struct {
	dataDir   string
	testAgent *agent.TestAgent
}

func NewConsulAgent(t *testing.T, port int) *ConsulAgent {
	dataDir, _ := ioutil.TempDir("./", "agent")
	hcl := `
		ports { 
			http = ` + strconv.Itoa(port) + `
		}
		data_dir = "` + dataDir + `"
	`
	testAgent := &agent.TestAgent{Name: t.Name(), DataDir: dataDir, HCL: hcl}
	testAgent.Start(t)

	consulAgent := &ConsulAgent{
		dataDir:   dataDir,
		testAgent: testAgent,
	}
	return consulAgent
}

func (consulAgent *ConsulAgent) Close() error {
	var err error

	err = consulAgent.testAgent.Shutdown()
	if err != nil {
		return err
	}
	return os.RemoveAll(consulAgent.dataDir)
}
