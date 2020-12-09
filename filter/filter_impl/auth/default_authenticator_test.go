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

package auth

import (
	"fmt"
	"net/url"
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
	"github.com/apache/dubbo-go/protocol/invocation"
)

func TestDefaultAuthenticator_Authenticate(t *testing.T) {
	secret := "dubbo-sk"
	access := "dubbo-ak"
	testurl, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	testurl.SetParam(constant.PARAMTER_SIGNATURE_ENABLE_KEY, "true")
	testurl.SetParam(constant.ACCESS_KEY_ID_KEY, access)
	testurl.SetParam(constant.SECRET_ACCESS_KEY_KEY, secret)
	parmas := []interface{}{"OK", struct {
		Name string
		Id   int64
	}{"YUYU", 1}}
	inv := invocation.NewRPCInvocation("test", parmas, nil)
	requestTime := strconv.Itoa(int(time.Now().Unix() * 1000))
	signature, _ := getSignature(testurl, inv, secret, requestTime)

	var authenticator = &DefaultAuthenticator{}

	invcation := invocation.NewRPCInvocation("test", parmas, map[string]interface{}{
		constant.REQUEST_SIGNATURE_KEY: signature,
		constant.CONSUMER:              "test",
		constant.REQUEST_TIMESTAMP_KEY: requestTime,
		constant.AK_KEY:                access,
	})
	err := authenticator.Authenticate(invcation, testurl)
	assert.Nil(t, err)
	// modify the params
	invcation = invocation.NewRPCInvocation("test", parmas[:1], map[string]interface{}{
		constant.REQUEST_SIGNATURE_KEY: signature,
		constant.CONSUMER:              "test",
		constant.REQUEST_TIMESTAMP_KEY: requestTime,
		constant.AK_KEY:                access,
	})
	err = authenticator.Authenticate(invcation, testurl)
	assert.NotNil(t, err)

}

func TestDefaultAuthenticator_Sign(t *testing.T) {
	authenticator := &DefaultAuthenticator{}
	testurl, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?application=test&interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	testurl.SetParam(constant.ACCESS_KEY_ID_KEY, "akey")
	testurl.SetParam(constant.SECRET_ACCESS_KEY_KEY, "skey")
	testurl.SetParam(constant.PARAMTER_SIGNATURE_ENABLE_KEY, "false")
	inv := invocation.NewRPCInvocation("test", []interface{}{"OK"}, nil)
	_ = authenticator.Sign(inv, testurl)
	assert.NotEqual(t, inv.AttachmentsByKey(constant.REQUEST_SIGNATURE_KEY, ""), "")
	assert.NotEqual(t, inv.AttachmentsByKey(constant.CONSUMER, ""), "")
	assert.NotEqual(t, inv.AttachmentsByKey(constant.REQUEST_TIMESTAMP_KEY, ""), "")
	assert.Equal(t, inv.AttachmentsByKey(constant.AK_KEY, ""), "akey")

}

func Test_getAccessKeyPairSuccess(t *testing.T) {
	testurl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.SECRET_ACCESS_KEY_KEY, "skey"),
		common.WithParamsValue(constant.ACCESS_KEY_ID_KEY, "akey"))
	invcation := invocation.NewRPCInvocation("MethodName", []interface{}{"OK"}, nil)
	_, e := getAccessKeyPair(invcation, testurl)
	assert.Nil(t, e)
}

func Test_getAccessKeyPairFailed(t *testing.T) {
	defer func() {
		e := recover()
		assert.NotNil(t, e)
	}()
	testurl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.ACCESS_KEY_ID_KEY, "akey"))
	invcation := invocation.NewRPCInvocation("MethodName", []interface{}{"OK"}, nil)
	_, e := getAccessKeyPair(invcation, testurl)
	assert.NotNil(t, e)
	testurl = common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.SECRET_ACCESS_KEY_KEY, "skey"),
		common.WithParamsValue(constant.ACCESS_KEY_ID_KEY, "akey"), common.WithParamsValue(constant.ACCESS_KEY_STORAGE_KEY, "dubbo"))
	_, e = getAccessKeyPair(invcation, testurl)

}

func Test_getSignatureWithinParams(t *testing.T) {
	testurl, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	testurl.SetParam(constant.PARAMTER_SIGNATURE_ENABLE_KEY, "true")
	inv := invocation.NewRPCInvocation("test", []interface{}{"OK"}, map[string]interface{}{
		"": "",
	})
	secret := "dubbo"
	current := strconv.Itoa(int(time.Now().Unix() * 1000))
	signature, _ := getSignature(testurl, inv, secret, current)
	requestString := fmt.Sprintf(constant.SIGNATURE_STRING_FORMAT,
		testurl.ColonSeparatedKey(), inv.MethodName(), secret, current)
	s, _ := SignWithParams(inv.Arguments(), requestString, secret)
	assert.False(t, IsEmpty(signature, false))
	assert.Equal(t, s, signature)
}

func Test_getSignature(t *testing.T) {
	testurl, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	testurl.SetParam(constant.PARAMTER_SIGNATURE_ENABLE_KEY, "false")
	inv := invocation.NewRPCInvocation("test", []interface{}{"OK"}, nil)
	secret := "dubbo"
	current := strconv.Itoa(int(time.Now().Unix() * 1000))
	signature, _ := getSignature(testurl, inv, secret, current)
	requestString := fmt.Sprintf(constant.SIGNATURE_STRING_FORMAT,
		testurl.ColonSeparatedKey(), inv.MethodName(), secret, current)
	s := Sign(requestString, secret)
	assert.False(t, IsEmpty(signature, false))
	assert.Equal(t, s, signature)
}
