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

package etcdv3

import (
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

func initRegistry(t *testing.T) *etcdV3Registry {

	regurl, err := common.NewURL("registry://127.0.0.1:2379", common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	if err != nil {
		t.Fatal(err)
	}

	reg, err := newETCDV3Registry(regurl)
	if err != nil {
		t.Fatal(err)
	}

	out := reg.(*etcdV3Registry)
	err = out.client.CleanKV()
	assert.NoError(t, err)
	return out
}

func (suite *RegistryTestSuite) TestRegister() {

	t := suite.T()

	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	reg := initRegistry(t)
	err := reg.Register(url)
	assert.NoError(t, err)
	children, _, err := reg.client.GetChildrenKVList("/dubbo/com.ikurento.user.UserProvider/providers")
	if err != nil {
		t.Fatal(err)
	}
	assert.Regexp(t, ".*dubbo%3A%2F%2F127.0.0.1%3A20000%2Fcom.ikurento.user.UserProvider%3Fanyhost%3Dtrue%26cluster%3Dmock", children)
	assert.NoError(t, err)
}

func (suite *RegistryTestSuite) TestSubscribe() {

	t := suite.T()
	regurl, _ := common.NewURL("registry://127.0.0.1:1111", common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	reg := initRegistry(t)
	//provider register
	err := reg.Register(url)
	if err != nil {
		t.Fatal(err)
	}

	//consumer register
	regurl.SetParam(constant.ROLE_KEY, strconv.Itoa(common.CONSUMER))
	reg2 := initRegistry(t)

	err = reg2.Register(url)
	assert.NoError(t, err)
	listener, err := reg2.DoSubscribe(url)
	if err != nil {
		t.Fatal(err)
	}

	serviceEvent, err := listener.Next()
	if err != nil {
		t.Fatal(err)
	}
	assert.Regexp(t, ".*ServiceEvent{Action{add}.*", serviceEvent.String())
}

func (suite *RegistryTestSuite) TestConsumerDestroy() {

	t := suite.T()
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	reg := initRegistry(t)
	_, err := reg.DoSubscribe(url)
	if err != nil {
		t.Fatal(err)
	}

	//listener.Close()
	time.Sleep(1e9)
	reg.Destroy()

	assert.Equal(t, false, reg.IsAvailable())

}

func (suite *RegistryTestSuite) TestProviderDestroy() {

	t := suite.T()
	reg := initRegistry(t)
	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))
	err := reg.Register(url)
	assert.NoError(t, err)

	//listener.Close()
	time.Sleep(1e9)
	reg.Destroy()
	assert.Equal(t, false, reg.IsAvailable())
}
