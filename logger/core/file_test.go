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

package core

import (
	"net/url"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestFileConfig_DefaultsAndOverrides(t *testing.T) {
	u := &common.URL{}
	// default values
	lj := FileConfig(u)
	if lj.Filename == "" || lj.Filename != "dubbo.log" {
		t.Fatalf("expected default filename dubbo.log, got %q", lj.Filename)
	}
	if lj.MaxSize != 1 || lj.MaxBackups != 1 || lj.MaxAge != 3 {
		t.Fatalf("unexpected defaults: size=%d backups=%d age=%d", lj.MaxSize, lj.MaxBackups, lj.MaxAge)
	}
	if !lj.LocalTime {
		t.Fatalf("expected default LocalTime true")
	}
	if !lj.Compress {
		t.Fatalf("expected default Compress true")
	}

	// overrides via params
	u = &common.URL{}
	u.ReplaceParams(url.Values{
		constant.LoggerFileNameKey:       []string{"app.log"},
		constant.LoggerFileNaxSizeKey:    []string{"8"},
		constant.LoggerFileMaxBackupsKey: []string{"4"},
		constant.LoggerFileMaxAgeKey:     []string{"9"},
		constant.LoggerFileLocalTimeKey:  []string{"false"},
		constant.LoggerFileCompressKey:   []string{"false"},
	})
	lj = FileConfig(u)
	if lj.Filename != "app.log" || lj.MaxSize != 8 || lj.MaxBackups != 4 || lj.MaxAge != 9 {
		t.Fatalf("unexpected overrides: %+v", lj)
	}
	if lj.LocalTime || lj.Compress {
		t.Fatalf("expected LocalTime=false and Compress=false, got LocalTime=%v Compress=%v", lj.LocalTime, lj.Compress)
	}
}
