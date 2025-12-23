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

package logger

import (
	"testing"
)

type mockLogger struct {
	level string
}

func (m *mockLogger) Debug(args ...any)                   {}
func (m *mockLogger) Debugf(template string, args ...any) {}
func (m *mockLogger) Info(args ...any)                    {}
func (m *mockLogger) Infof(template string, args ...any)  {}
func (m *mockLogger) Warn(args ...any)                    {}
func (m *mockLogger) Warnf(template string, args ...any)  {}
func (m *mockLogger) Error(args ...any)                   {}
func (m *mockLogger) Errorf(template string, args ...any) {}
func (m *mockLogger) Fatal(args ...any)                   {}
func (m *mockLogger) Fatalf(fmt string, args ...any)      {}

func (m *mockLogger) SetLoggerLevel(level string) bool {
	m.level = level
	return true
}

func TestSetAndGetLogger(t *testing.T) {
	m := &mockLogger{}
	SetLogger(m)
	if GetLogger() != m {
		t.Fatalf("GetLogger should return the logger set by SetLogger")
	}
}

func TestSetLoggerLevel(t *testing.T) {
	m := &mockLogger{}
	SetLogger(m)
	ok := SetLoggerLevel("warn")
	if !ok {
		t.Fatalf("expected SetLoggerLevel to return true for OpsLogger")
	}
	if m.level != "warn" {
		t.Fatalf("expected level 'warn', got %q", m.level)
	}
}

// simpleLogger implements Logger but NOT OpsLogger
type simpleLogger struct{}

func (s *simpleLogger) Debug(args ...any)                   {}
func (s *simpleLogger) Debugf(template string, args ...any) {}
func (s *simpleLogger) Info(args ...any)                    {}
func (s *simpleLogger) Infof(template string, args ...any)  {}
func (s *simpleLogger) Warn(args ...any)                    {}
func (s *simpleLogger) Warnf(template string, args ...any)  {}
func (s *simpleLogger) Error(args ...any)                   {}
func (s *simpleLogger) Errorf(template string, args ...any) {}
func (s *simpleLogger) Fatal(args ...any)                   {}
func (s *simpleLogger) Fatalf(fmt string, args ...any)      {}

func TestSetLoggerLevel_NotOpsLogger(t *testing.T) {
	s := &simpleLogger{}
	SetLogger(s)
	ok := SetLoggerLevel("debug")
	if ok {
		t.Fatalf("expected SetLoggerLevel to return false for non-OpsLogger")
	}
}
