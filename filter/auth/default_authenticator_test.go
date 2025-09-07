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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestDefaultAuthenticator_Authenticate(t *testing.T) {
	secret := "dubbo-sk"
	access := "dubbo-ak"
	testUrl, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	testUrl.SetParam(constant.ParameterSignatureEnableKey, "true")
	testUrl.SetParam(constant.AccessKeyIDKey, access)
	testUrl.SetParam(constant.SecretAccessKeyKey, secret)
	params := []any{"OK", struct {
		Name string
		ID   int64
	}{"YUYU", 1}}
	inv := invocation.NewRPCInvocation("test", params, nil)
	requestTime := strconv.Itoa(int(time.Now().Unix() * 1000))
	signature, _ := getSignature(testUrl, inv, secret, requestTime)

	authenticator = &defaultAuthenticator{}

	rpcInvocation := invocation.NewRPCInvocation("test", params, map[string]any{
		constant.RequestSignatureKey: signature,
		constant.Consumer:            "test",
		constant.RequestTimestampKey: requestTime,
		constant.AKKey:               access,
	})
	err := authenticator.Authenticate(rpcInvocation, testUrl)
	assert.Nil(t, err)
	// modify the params
	rpcInvocation = invocation.NewRPCInvocation("test", params[:1], map[string]any{
		constant.RequestSignatureKey: signature,
		constant.Consumer:            "test",
		constant.RequestTimestampKey: requestTime,
		constant.AKKey:               access,
	})
	err = authenticator.Authenticate(rpcInvocation, testUrl)
	assert.NotNil(t, err)
}

func TestDefaultAuthenticator_Sign(t *testing.T) {
	authenticator = &defaultAuthenticator{}
	testUrl, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?application=test&interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	testUrl.SetParam(constant.AccessKeyIDKey, "akey")
	testUrl.SetParam(constant.SecretAccessKeyKey, "skey")
	testUrl.SetParam(constant.ParameterSignatureEnableKey, "false")
	inv := invocation.NewRPCInvocation("test", []any{"OK"}, nil)
	_ = authenticator.Sign(inv, testUrl)
	assert.NotEqual(t, inv.GetAttachmentWithDefaultValue(constant.RequestSignatureKey, ""), "")
	assert.NotEqual(t, inv.GetAttachmentWithDefaultValue(constant.Consumer, ""), "")
	assert.NotEqual(t, inv.GetAttachmentWithDefaultValue(constant.RequestTimestampKey, ""), "")
	assert.Equal(t, inv.GetAttachmentWithDefaultValue(constant.AKKey, ""), "akey")
}

func TestDefaultAuthenticator_SignNotAccessKeyIDKeyError(t *testing.T) {
	authenticator = &defaultAuthenticator{}
	testUrl, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?application=test&interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	testUrl.SetParam(constant.SecretAccessKeyKey, "skey")
	testUrl.SetParam(constant.ParameterSignatureEnableKey, "false")
	inv := invocation.NewRPCInvocation("test", []any{"OK"}, nil)
	err := authenticator.Sign(inv, testUrl)
	assert.NotNil(t, err)
}

func Test_getAccessKeyPairSuccess(t *testing.T) {
	testUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.SecretAccessKeyKey, "skey"),
		common.WithParamsValue(constant.AccessKeyIDKey, "akey"))
	rpcInvocation := invocation.NewRPCInvocation("MethodName", []any{"OK"}, nil)
	_, e := getAccessKeyPair(rpcInvocation, testUrl)
	assert.Nil(t, e)
}

func Test_getAccessKeyPairFailed(t *testing.T) {
	testUrl := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.AccessKeyIDKey, "akey"))
	rpcInvocation := invocation.NewRPCInvocation("MethodName", []any{"OK"}, nil)
	_, e := getAccessKeyPair(rpcInvocation, testUrl)
	assert.NotNil(t, e)
	testUrl = common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.SecretAccessKeyKey, "skey"),
		common.WithParamsValue(constant.AccessKeyIDKey, "akey"),
		common.WithParamsValue(constant.AccessKeyStorageKey, "dubbo"))
	_, e = getAccessKeyPair(rpcInvocation, testUrl)
	assert.NotNil(t, e)
	assert.Contains(t, e.Error(), "accessKeyStorages for dubbo is not existing")
}

func Test_getSignatureWithinParams(t *testing.T) {
	testUrl, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	testUrl.SetParam(constant.ParameterSignatureEnableKey, "true")
	inv := invocation.NewRPCInvocation("test", []any{"OK"}, map[string]any{
		"": "",
	})
	secret := "dubbo"
	current := strconv.Itoa(int(time.Now().Unix() * 1000))
	signature, _ := getSignature(testUrl, inv, secret, current)
	requestString := fmt.Sprintf(constant.SignatureStringFormat,
		testUrl.ColonSeparatedKey(), inv.MethodName(), secret, current)
	s, _ := SignWithParams(inv.Arguments(), requestString, secret)
	assert.False(t, IsEmpty(signature, false))
	assert.Equal(t, s, signature)
}

func Test_getSignature(t *testing.T) {
	testUrl, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	testUrl.SetParam(constant.ParameterSignatureEnableKey, "false")
	inv := invocation.NewRPCInvocation("test", []any{"OK"}, nil)
	secret := "dubbo"
	current := strconv.Itoa(int(time.Now().Unix() * 1000))
	signature, _ := getSignature(testUrl, inv, secret, current)
	requestString := fmt.Sprintf(constant.SignatureStringFormat,
		testUrl.ColonSeparatedKey(), inv.MethodName(), secret, current)
	s := Sign(requestString, secret)
	assert.False(t, IsEmpty(signature, false))
	assert.Equal(t, s, signature)
}
