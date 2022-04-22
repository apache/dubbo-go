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
	"sync"
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

var (
	authenticatorOnce sync.Once
	authenticator     *defaultAuthenticator
)

func init() {
	extension.SetAuthenticator(constant.DefaultAuthenticator, newDefaultAuthenticator)
}

// defaultAuthenticator is the default implementation of Authenticator
type defaultAuthenticator struct{}

func newDefaultAuthenticator() filter.Authenticator {
	if authenticator == nil {
		authenticatorOnce.Do(func() {
			authenticator = &defaultAuthenticator{}
		})
	}
	return authenticator
}

// Sign adds the signature to the invocation
func (authenticator *defaultAuthenticator) Sign(invocation protocol.Invocation, url *common.URL) error {
	currentTimeMillis := strconv.Itoa(int(time.Now().Unix() * 1000))

	consumer := url.GetParam(constant.ApplicationKey, "")
	accessKeyPair, err := getAccessKeyPair(invocation, url)
	if err != nil {
		return errors.New("get accesskey pair failed, cause: " + err.Error())
	}
	inv := invocation.(*invocation_impl.RPCInvocation)
	signature, err := getSignature(url, invocation, accessKeyPair.SecretKey, currentTimeMillis)
	if err != nil {
		return err
	}
	inv.SetAttachment(constant.RequestSignatureKey, signature)
	inv.SetAttachment(constant.RequestTimestampKey, currentTimeMillis)
	inv.SetAttachment(constant.AKKey, accessKeyPair.AccessKey)
	inv.SetAttachment(constant.Consumer, consumer)
	return nil
}

// getSignature
// get signature by the metadata and params of the invocation
func getSignature(url *common.URL, invocation protocol.Invocation, secrectKey string, currentTime string) (string, error) {
	requestString := fmt.Sprintf(constant.SignatureStringFormat,
		url.ColonSeparatedKey(), invocation.MethodName(), secrectKey, currentTime)
	var signature string
	if parameterEncrypt := url.GetParamBool(constant.ParameterSignatureEnableKey, false); parameterEncrypt {
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
func (authenticator *defaultAuthenticator) Authenticate(invocation protocol.Invocation, url *common.URL) error {
	accessKeyId := invocation.GetAttachmentWithDefaultValue(constant.AKKey, "")

	requestTimestamp := invocation.GetAttachmentWithDefaultValue(constant.RequestTimestampKey, "")
	originSignature := invocation.GetAttachmentWithDefaultValue(constant.RequestSignatureKey, "")
	consumer := invocation.GetAttachmentWithDefaultValue(constant.Consumer, "")
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
	accesskeyStorage := extension.GetAccessKeyStorages(url.GetParam(constant.AccessKeyStorageKey, constant.DefaultAccessKeyStorage))
	accessKeyPair := accesskeyStorage.GetAccessKeyPair(invocation, url)
	if accessKeyPair == nil || IsEmpty(accessKeyPair.AccessKey, false) || IsEmpty(accessKeyPair.SecretKey, true) {
		return nil, errors.New("accessKeyId or secretAccessKey not found")
	} else {
		return accessKeyPair, nil
	}
}

func doAuthWork(url *common.URL, do func(filter.Authenticator) error) error {
	shouldAuth := url.GetParamBool(constant.ServiceAuthKey, false)
	if shouldAuth {
		authenticator, exist := extension.GetAuthenticator(url.GetParam(constant.AuthenticatorKey, constant.DefaultAuthenticator))
		if exist {
			return do(authenticator)

		} else {
			return errors.New("Authenticator for " + constant.ServiceAuthKey + " is not existing, make sure you have import the package.")
		}
	}
	return nil
}
