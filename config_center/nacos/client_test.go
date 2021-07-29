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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestNewNacosClient(t *testing.T) {

	nacosURL := "registry://127.0.0.1:8848"
	registryUrl, _ := common.NewURL(nacosURL)

	c := &nacosDynamicConfiguration{
		url:  registryUrl,
		done: make(chan struct{}),
	}
	err := ValidateNacosClient(c)
	assert.NoError(t, err)
	c.wg.Add(1)
	go HandleClientRestart(c)
	go func() {
		// c.configClient.Close() and <-c.configClient.Done() have order requirements.
		// If c.configClient.Close() is called first.It is possible that "go HandleClientRestart(c)"
		// sets c.configClient to nil before calling c.configClient.Done().
		time.Sleep(time.Second)
		c.client.Close()
	}()
	//<-c.client.Done()
	c.Destroy()
}

func TestSetNacosClient(t *testing.T) {
	nacosURL := "registry://127.0.0.1:8848"
	registryUrl, _ := common.NewURL(nacosURL)

	c := &nacosDynamicConfiguration{
		url:  registryUrl,
		done: make(chan struct{}),
	}

	err := ValidateNacosClient(c)
	assert.NoError(t, err)
	c.wg.Add(1)
	go HandleClientRestart(c)
	go func() {
		// c.configClient.Close() and <-c.configClient.Done() have order requirements.
		// If c.configClient.Close() is called first.It is possible that "go HandleClientRestart(c)"
		// sets c.configClient to nil before calling c.configClient.Done().
		time.Sleep(time.Second)
		c.client.Close()
	}()
	c.Destroy()
}

func TestNewNacosClient_connectError(t *testing.T) {
	nacosURL := "registry://127.0.0.1:8848"
	registryUrl, err := common.NewURL(nacosURL)
	assert.NoError(t, err)
	c := &nacosDynamicConfiguration{
		url:  registryUrl,
		done: make(chan struct{}),
	}
	err = ValidateNacosClient(c)
	assert.NoError(t, err)
	c.wg.Add(1)
	go HandleClientRestart(c)
	go func() {
		// c.configClient.Close() and <-c.configClient.Done() have order requirements.
		// If c.configClient.Close() is called first.It is possible that "go HandleClientRestart(c)"
		// sets c.configClient to nil before calling c.configClient.Done().
		time.Sleep(time.Second)
		c.client.Close()
	}()
	// <-c.client.Done()
	// let configClient do retry
	time.Sleep(5 * time.Second)
	c.Destroy()
}
