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

func TestInstantiateZap_TextFormat(t *testing.T) {
	u := &common.URL{}
	u.ReplaceParams(url.Values{
		constant.LoggerLevelKey:    []string{"info"},
		constant.LoggerAppenderKey: []string{"console"},
		constant.LoggerFormatKey:   []string{"text"},
	})
	lg, err := instantiate(u)
	if err != nil || lg == nil {
		t.Fatalf("expected zap logger with text format, err=%v", err)
	}
}

func TestInstantiateZap_DefaultFormat(t *testing.T) {
	u := &common.URL{}
	u.ReplaceParams(url.Values{
		constant.LoggerLevelKey:    []string{"info"},
		constant.LoggerAppenderKey: []string{"console"},
		constant.LoggerFormatKey:   []string{"unknown-format"},
	})
	lg, err := instantiate(u)
	if err != nil || lg == nil {
		t.Fatalf("expected zap logger with default format fallback, err=%v", err)
	}
}

func TestInstantiateZap_ConsoleAndFileAppender(t *testing.T) {
	u := &common.URL{}
	u.ReplaceParams(url.Values{
		constant.LoggerLevelKey:       []string{"debug"},
		constant.LoggerAppenderKey:    []string{"console,file"},
		constant.LoggerFormatKey:      []string{"json"},
		constant.LoggerFileNameKey:    []string{"test_combined.log"},
		constant.LoggerFileNaxSizeKey: []string{"1"},
	})
	lg, err := instantiate(u)
	if err != nil || lg == nil {
		t.Fatalf("expected zap logger with console and file appenders, err=%v", err)
	}
}

func TestInstantiateZap_InvalidLevel(t *testing.T) {
	u := &common.URL{}
	u.ReplaceParams(url.Values{
		constant.LoggerLevelKey:    []string{"not-a-valid-level"},
		constant.LoggerAppenderKey: []string{"console"},
	})
	lg, err := instantiate(u)
	if err == nil {
		t.Fatalf("expected error for invalid level, got nil")
	}
	if lg != nil {
		t.Fatalf("expected nil logger for invalid level, got %v", lg)
	}
}

func TestNewDefault(t *testing.T) {
	lg := NewDefault()
	if lg == nil {
		t.Fatalf("expected non-nil default logger")
	}
	if lg.Logger == nil {
		t.Fatalf("expected non-nil underlying logger")
	}
}

func TestEncoderConfig(t *testing.T) {
	ec := encoderConfig()
	if ec.MessageKey != "msg" {
		t.Fatalf("expected MessageKey 'msg', got %q", ec.MessageKey)
	}
	if ec.LevelKey != "level" {
		t.Fatalf("expected LevelKey 'level', got %q", ec.LevelKey)
	}
	if ec.TimeKey != "time" {
		t.Fatalf("expected TimeKey 'time', got %q", ec.TimeKey)
	}
	if ec.CallerKey != "line" {
		t.Fatalf("expected CallerKey 'line', got %q", ec.CallerKey)
	}
	if ec.NameKey != "logger" {
		t.Fatalf("expected NameKey 'logger', got %q", ec.NameKey)
	}
	if ec.StacktraceKey != "stacktrace" {
		t.Fatalf("expected StacktraceKey 'stacktrace', got %q", ec.StacktraceKey)
	}
}
