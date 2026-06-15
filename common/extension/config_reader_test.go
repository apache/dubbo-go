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

package extension

import (
	"bytes"
	"testing"
)

import (
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
)

type testConfigReader struct{}

func (*testConfigReader) ReadConsumerConfig(*bytes.Buffer) error {
	return nil
}

func (*testConfigReader) ReadProviderConfig(*bytes.Buffer) error {
	return nil
}

func TestConfigReaderRegistry(t *testing.T) {
	t.Cleanup(func() {
		configReaders.Unregister("config-reader-test")
		defaults.Unregister("config-reader-module")
	})

	SetConfigReaders("config-reader-test", func() commonCfg.ConfigReader {
		return &testConfigReader{}
	})
	SetDefaultConfigReader("config-reader-module", "config-reader-test")

	if _, ok := GetConfigReaders("config-reader-test").(*testConfigReader); !ok {
		t.Fatal("expected registered config reader")
	}
	if got := GetDefaultConfigReader()["config-reader-module"]; got != "config-reader-test" {
		t.Fatalf("expected default config reader, got %q", got)
	}
}
