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

package zap

import (
	"net/url"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestInstantiateZap_ConsoleJsonAndLevel(t *testing.T) {
	u := &common.URL{}
	u.ReplaceParams(url.Values{
		constant.LoggerLevelKey:    []string{"info"},
		constant.LoggerAppenderKey: []string{"console"},
		constant.LoggerFormatKey:   []string{"json"},
	})
	lg, err := instantiate(u)
	if err != nil || lg == nil {
		t.Fatalf("expected zap logger, err=%v", err)
	}
}

func TestInstantiateZap_FileAppender(t *testing.T) {
	u := &common.URL{}
	u.ReplaceParams(url.Values{
		constant.LoggerLevelKey:       []string{"debug"},
		constant.LoggerAppenderKey:    []string{"file"},
		constant.LoggerFileNameKey:    []string{"test.log"},
		constant.LoggerFileNaxSizeKey: []string{"1"},
	})
	lg, err := instantiate(u)
	if err != nil || lg == nil {
		t.Fatalf("expected zap file logger, err=%v", err)
	}
}
