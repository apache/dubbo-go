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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

// Sign gets a signature string with given bytes
func Sign(metadata, key string) string {
	return doSign([]byte(metadata), key)
}

// SignWithParams returns a signature with giving params and metadata.
func SignWithParams(params []interface{}, metadata, key string) (string, error) {
	if len(params) == 0 {
		return Sign(metadata, key), nil
	}

	data := append(params, metadata)
	if bytes, err := toBytes(data); err != nil {
		// TODO
		return "", errors.New("data convert to bytes failed")
	} else {
		return doSign(bytes, key), nil
	}
}

func toBytes(data []interface{}) ([]byte, error) {
	if bytes, err := json.Marshal(data); err != nil {
		return nil, errors.New("")
	} else {
		return bytes, nil
	}
}

func doSign(bytes []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	if _, err := mac.Write(bytes); err != nil {
		logger.Error(err)
	}
	signature := mac.Sum(nil)
	return base64.URLEncoding.EncodeToString(signature)
}

// IsEmpty verify whether the inputted string is empty
func IsEmpty(s string, allowSpace bool) bool {
	if len(s) == 0 {
		return true
	}
	if !allowSpace {
		return strings.TrimSpace(s) == ""
	}
	return false
}
