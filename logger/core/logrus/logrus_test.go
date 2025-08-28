// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logrus

import (
	"net/url"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestInstantiateLogrus_ConsoleAndInvalidLevelFallback(t *testing.T) {
	u := &common.URL{}
	u.ReplaceParams(url.Values{
		constant.LoggerLevelKey:    []string{"not-a-level"},
		constant.LoggerAppenderKey: []string{"console"},
		constant.LoggerFormatKey:   []string{"text"},
	})
	lg, err := instantiate(u)
	// Invalid level returns an error, but logger should still be created with fallback level
	if lg == nil {
		t.Fatalf("expected non-nil logger even if level invalid, err=%v", err)
	}
}

func TestInstantiateLogrus_FileAppenderAndJson(t *testing.T) {
	u := &common.URL{}
	u.ReplaceParams(url.Values{
		constant.LoggerLevelKey:       []string{"warn"},
		constant.LoggerAppenderKey:    []string{"file"},
		constant.LoggerFormatKey:      []string{"json"},
		constant.LoggerFileNameKey:    []string{"test.log"},
		constant.LoggerFileNaxSizeKey: []string{"1"},
	})
	lg, err := instantiate(u)
	if err != nil || lg == nil {
		t.Fatalf("expected logrus file logger, err=%v", err)
	}
}
