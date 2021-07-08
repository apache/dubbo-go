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
	"errors"
	"fmt"
	"strconv"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	invocation_impl "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func init() {
	extension.SetAuthenticator(constant.DEFAULT_AUTHENTICATOR, func() filter.Authenticator {
		return &DefaultAuthenticator{}
	})
}

// DefaultAuthenticator is the default implementation of Authenticator
type DefaultAuthenticator struct{}

// Sign adds the signature to the invocation
func (authenticator *DefaultAuthenticator) Sign(invocation protocol.Invocation, url *common.URL) error {
	currentTimeMillis := strconv.Itoa(int(time.Now().Unix() * 1000))

	consumer := url.GetParam(constant.APPLICATION_KEY, "")
	accessKeyPair, err := getAccessKeyPair(invocation, url)
	if err != nil {
		return errors.New("get accesskey pair failed, cause: " + err.Error())
	}
	inv := invocation.(*invocation_impl.RPCInvocation)
	signature, err := getSignature(url, invocation, accessKeyPair.SecretKey, currentTimeMillis)
	if err != nil {
		return err
	}
	inv.SetAttachments(constant.REQUEST_SIGNATURE_KEY, signature)
	inv.SetAttachments(constant.REQUEST_TIMESTAMP_KEY, currentTimeMillis)
	inv.SetAttachments(constant.AK_KEY, accessKeyPair.AccessKey)
	inv.SetAttachments(constant.CONSUMER, consumer)
	return nil
}

// getSignature
// get signature by the metadata and params of the invocation
func getSignature(url *common.URL, invocation protocol.Invocation, secrectKey string, currentTime string) (string, error) {
	requestString := fmt.Sprintf(constant.SIGNATURE_STRING_FORMAT,
		url.ColonSeparatedKey(), invocation.MethodName(), secrectKey, currentTime)
	var signature string
	if parameterEncrypt := url.GetParamBool(constant.PARAMETER_SIGNATURE_ENABLE_KEY, false); parameterEncrypt {
		var err error
		if signature, err = SignWithParams(invocation.Arguments(), requestString, secrectKey); err != nil {
			// TODO
			return "", errors.New("sign the request with params failed, cause:" + err.Error())
		}
	} else {
		signature = Sign(requestString, secrectKey)
	}

	return signature, nil
}

// Authenticate verifies whether the signature sent by the requester is correct
func (authenticator *DefaultAuthenticator) Authenticate(invocation protocol.Invocation, url *common.URL) error {
	accessKeyId := invocation.AttachmentsByKey(constant.AK_KEY, "")

	requestTimestamp := invocation.AttachmentsByKey(constant.REQUEST_TIMESTAMP_KEY, "")
	originSignature := invocation.AttachmentsByKey(constant.REQUEST_SIGNATURE_KEY, "")
	consumer := invocation.AttachmentsByKey(constant.CONSUMER, "")
	if IsEmpty(accessKeyId, false) || IsEmpty(consumer, false) ||
		IsEmpty(requestTimestamp, false) || IsEmpty(originSignature, false) {
		return errors.New("failed to authenticate your ak/sk, maybe the consumer has not enabled the auth")
	}

	accessKeyPair, err := getAccessKeyPair(invocation, url)
	if err != nil {
		return errors.New("failed to authenticate , can't load the accessKeyPair")
	}

	computeSignature, err := getSignature(url, invocation, accessKeyPair.SecretKey, requestTimestamp)
	if err != nil {
		return err
	}
	if success := computeSignature == originSignature; !success {
		return errors.New("failed to authenticate, signature is not correct")
	}
	return nil
}

func getAccessKeyPair(invocation protocol.Invocation, url *common.URL) (*filter.AccessKeyPair, error) {
	accesskeyStorage := extension.GetAccessKeyStorages(url.GetParam(constant.ACCESS_KEY_STORAGE_KEY, constant.DEFAULT_ACCESS_KEY_STORAGE))
	accessKeyPair := accesskeyStorage.GetAccessKeyPair(invocation, url)
	if accessKeyPair == nil || IsEmpty(accessKeyPair.AccessKey, false) || IsEmpty(accessKeyPair.SecretKey, true) {
		return nil, errors.New("accessKeyId or secretAccessKey not found")
	} else {
		return accessKeyPair, nil
	}
}

func doAuthWork(url *common.URL, do func(filter.Authenticator) error) error {
	shouldAuth := url.GetParamBool(constant.SERVICE_AUTH_KEY, false)
	if shouldAuth {
		authenticator := extension.GetAuthenticator(url.GetParam(constant.AUTHENTICATOR_KEY, constant.DEFAULT_AUTHENTICATOR))
		return do(authenticator)
	}
	return nil
}
