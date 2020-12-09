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

package nacos

import (
	"strings"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
)

func TestNewNacosClient(t *testing.T) {
	server := mockCommonNacosServer()
	nacosURL := strings.ReplaceAll(server.URL, "http", "registry")
	registryUrl, _ := common.NewURL(nacosURL)
	c := &nacosDynamicConfiguration{
		url:  registryUrl,
		done: make(chan struct{}),
	}
	err := ValidateNacosClient(c, WithNacosName(nacosClientName))
	assert.NoError(t, err)
	c.wg.Add(1)
	go HandleClientRestart(c)
	go func() {
		// c.client.Close() and <-c.client.Done() have order requirements.
		// If c.client.Close() is called first.It is possible that "go HandleClientRestart(c)"
		// sets c.client to nil before calling c.client.Done().
		time.Sleep(time.Second)
		c.client.Close()
	}()
	<-c.client.Done()
	c.Destroy()
}

func TestSetNacosClient(t *testing.T) {
	server := mockCommonNacosServer()
	nacosURL := "registry://" + server.Listener.Addr().String()
	registryUrl, _ := common.NewURL(nacosURL)
	c := &nacosDynamicConfiguration{
		url:  registryUrl,
		done: make(chan struct{}),
	}
	var client *NacosClient
	client = &NacosClient{
		name:       nacosClientName,
		NacosAddrs: []string{nacosURL},
		Timeout:    15 * time.Second,
		exit:       make(chan struct{}),
		onceClose: func() {
			close(client.exit)
		},
	}
	c.SetNacosClient(client)
	err := ValidateNacosClient(c, WithNacosName(nacosClientName))
	assert.NoError(t, err)
	c.wg.Add(1)
	go HandleClientRestart(c)
	go func() {
		// c.client.Close() and <-c.client.Done() have order requirements.
		// If c.client.Close() is called first.It is possible that "go HandleClientRestart(c)"
		// sets c.client to nil before calling c.client.Done().
		time.Sleep(time.Second)
		c.client.Close()
	}()
	<-c.client.Done()
	c.Destroy()
}

func TestNewNacosClient_connectError(t *testing.T) {
	nacosURL := "registry://127.0.0.1:8888"
	registryUrl, err := common.NewURL(nacosURL)
	assert.NoError(t, err)
	c := &nacosDynamicConfiguration{
		url:  registryUrl,
		done: make(chan struct{}),
	}
	err = ValidateNacosClient(c, WithNacosName(nacosClientName))
	assert.NoError(t, err)
	c.wg.Add(1)
	go HandleClientRestart(c)
	go func() {
		// c.client.Close() and <-c.client.Done() have order requirements.
		// If c.client.Close() is called first.It is possible that "go HandleClientRestart(c)"
		// sets c.client to nil before calling c.client.Done().
		time.Sleep(time.Second)
		c.client.Close()
	}()
	<-c.client.Done()
	// let client do retry
	time.Sleep(5 * time.Second)
	c.Destroy()
}
