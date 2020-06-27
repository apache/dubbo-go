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

package zookeeper

import (
	"strconv"
	"testing"
)

import (
	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/registry"
)

var testName = "test"

func prepareData(t *testing.T) *zk.TestCluster {
	ts, err := zk.StartTestCluster(1, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, ts.Servers[0])
	address := "127.0.0.1:" + strconv.Itoa(ts.Servers[0].Port)

	config.GetBaseConfig().ServiceDiscoveries[testName] = &config.ServiceDiscoveryConfig{
		Protocol:  "zookeeper",
		RemoteRef: "test",
	}

	config.GetBaseConfig().Remotes[testName] = &config.RemoteConfig{
		Address:    address,
		TimeoutStr: "10s",
	}
	return ts
}

func TestNewZookeeperServiceDiscovery(t *testing.T) {
	name := "zookeeper1"
	_, err := newZookeeperServiceDiscovery(name)

	// the ServiceDiscoveryConfig not found
	assert.NotNil(t, err)

	sdc := &config.ServiceDiscoveryConfig{
		Protocol:  "zookeeper",
		RemoteRef: "mock",
	}
	config.GetBaseConfig().ServiceDiscoveries[name] = sdc
	_, err = newZookeeperServiceDiscovery(name)

	// RemoteConfig not found
	assert.NotNil(t, err)
}

func TestCURDZookeeperServiceDiscovery(t *testing.T) {
	ts := prepareData(t)
	defer ts.Stop()
	sd, err := newZookeeperServiceDiscovery(testName)
	assert.Nil(t, err)
	md := make(map[string]string)
	md["t1"] = "test1"
	err = sd.Register(&registry.DefaultServiceInstance{
		Id:          "testId",
		ServiceName: testName,
		Host:        "127.0.0.1",
		Port:        2233,
		Enable:      true,
		Healthy:     true,
		Metadata:    md,
	})
	assert.Nil(t, err)
	tests := sd.GetInstances(testName)
	assert.Equal(t, tests[0].GetId(), "127.0.0.1:2233")
	err = sd.Unregister(&registry.DefaultServiceInstance{
		Id:          "testId",
		ServiceName: testName,
		Host:        "127.0.0.1",
		Port:        2233,
		Enable:      true,
		Healthy:     true,
		Metadata:    nil,
	})
	assert.Nil(t, err)
}
