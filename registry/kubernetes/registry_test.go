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

package kubernetes

import (
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

func (s *KubernetesRegistryTestSuite) TestRegister() {

	t := s.T()

	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	err := s.registry.Register(url)
	assert.NoError(t, err)
	_, _, err = s.registry.client.GetChildren("/dubbo/com.ikurento.user.UserProvider/providers")
	if err != nil {
		t.Fatal(err)
	}
}

func (s *KubernetesRegistryTestSuite) TestSubscribe() {

	t := s.T()

	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	listener, err := s.registry.DoSubscribe(&url)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := s.registry.Register(url)
		if err != nil {
			t.Fatal(err)
		}
	}()

	serviceEvent, err := listener.Next()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("got event %s", serviceEvent)
}

func (s *KubernetesRegistryTestSuite) TestConsumerDestroy() {

	t := s.T()
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	_, err := s.registry.DoSubscribe(&url)
	if err != nil {
		t.Fatal(err)
	}

	//listener.Close()
	time.Sleep(1e9)
	s.registry.Destroy()

	assert.Equal(t, false, s.registry.IsAvailable())

}

func (s *KubernetesRegistryTestSuite) TestProviderDestroy() {

	t := s.T()

	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))
	err := s.registry.Register(url)
	assert.NoError(t, err)

	//listener.Close()
	time.Sleep(1e9)
	s.registry.Destroy()
	assert.Equal(t, false, s.registry.IsAvailable())
}
