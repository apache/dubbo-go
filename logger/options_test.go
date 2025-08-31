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

func TestNewOptionsAndWithers(t *testing.T) {
	opts := NewOptions(
		WithZap(),
		WithLevel("debug"),
		WithFormat("json"),
		WithAppender("console,file"),
		WithFileName("app.log"),
		WithFileMaxSize(10),
		WithFileMaxBackups(5),
		WithFileMaxAge(7),
		WithFileCompress(),
	)

	if opts.Logger.Driver != "zap" {
		t.Fatalf("expected driver zap, got %s", opts.Logger.Driver)
	}
	if opts.Logger.Level != "debug" {
		t.Fatalf("expected level debug, got %s", opts.Logger.Level)
	}
	if opts.Logger.Format != "json" {
		t.Fatalf("expected format json, got %s", opts.Logger.Format)
	}
	if opts.Logger.Appender != "console,file" {
		t.Fatalf("expected appender console,file, got %s", opts.Logger.Appender)
	}
	if opts.Logger.File.Name != "app.log" {
		t.Fatalf("expected file name app.log, got %s", opts.Logger.File.Name)
	}
	if opts.Logger.File.MaxSize != 10 {
		t.Fatalf("expected file max size 10, got %d", opts.Logger.File.MaxSize)
	}
	if opts.Logger.File.MaxBackups != 5 {
		t.Fatalf("expected file max backups 5, got %d", opts.Logger.File.MaxBackups)
	}
	if opts.Logger.File.MaxAge != 7 {
		t.Fatalf("expected file max age 7, got %d", opts.Logger.File.MaxAge)
	}
	if opts.Logger.File.Compress == nil || *opts.Logger.File.Compress != true {
		t.Fatalf("expected file compress true, got %+v", opts.Logger.File.Compress)
	}
}
