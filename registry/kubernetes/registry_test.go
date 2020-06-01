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
	"strconv"
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

	r := s.initRegistry()
	defer r.Destroy()

	url, _ := common.NewURL(
		"dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
		common.WithMethods([]string{"GetUser", "AddUser"}),
	)

	err := r.Register(url)
	assert.NoError(t, err)
	_, _, err = r.client.GetChildren("/dubbo/com.ikurento.user.UserProvider/providers")
	if err != nil {
		t.Fatal(err)
	}
}

func (s *KubernetesRegistryTestSuite) TestSubscribe() {

	t := s.T()

	r := s.initRegistry()
	defer r.Destroy()

	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	listener, err := r.DoSubscribe(&url)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1e9)

	go func() {
		registerErr := r.Register(url)
		if registerErr != nil {
			t.Fatal(registerErr)
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

	r := s.initRegistry()

	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
		common.WithMethods([]string{"GetUser", "AddUser"}))

	_, err := r.DoSubscribe(&url)
	if err != nil {
		t.Fatal(err)
	}

	//listener.Close()
	time.Sleep(1e9)
	r.Destroy()

	assert.Equal(t, false, r.IsAvailable())

}

func (s *KubernetesRegistryTestSuite) TestProviderDestroy() {

	t := s.T()

	r := s.initRegistry()

	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider",
		common.WithParamsValue(constant.CLUSTER_KEY, "mock"),
		common.WithMethods([]string{"GetUser", "AddUser"}))
	err := r.Register(url)
	assert.NoError(t, err)

	time.Sleep(1e9)
	r.Destroy()
	assert.Equal(t, false, r.IsAvailable())
}

func (s *KubernetesRegistryTestSuite) TestNewRegistry() {

	t := s.T()

	regUrl, err := common.NewURL("registry://127.0.0.1:443",
		common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	if err != nil {
		t.Fatal(err)
	}
	_, err = newKubernetesRegistry(&regUrl)
	if err == nil {
		t.Fatal("not in cluster, should be a err")
	}
}

func (s *KubernetesRegistryTestSuite) TestHandleClientRestart() {

	r := s.initRegistry()
	r.WaitGroup().Add(1)
	go r.HandleClientRestart()
	time.Sleep(timeSecondDuration(1))
	r.client.Close()
}
