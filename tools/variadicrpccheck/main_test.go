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

package main

import (
	"os"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainUsesRunExitCode(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, goModFileName, variadicCheckModuleContent)
	writeTempFile(t, dir, serviceFileName, `package sample

func Echo(name string) string {
	return name
}
`)

	oldExitFunc := exitFunc
	oldArgs := os.Args
	oldWd, err := os.Getwd()
	require.NoError(t, err)

	var gotCode int
	exitFunc = func(code int) {
		gotCode = code
	}
	os.Args = []string{"variadicrpccheck", "./..."}
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		exitFunc = oldExitFunc
		os.Args = oldArgs
		_ = os.Chdir(oldWd)
	})

	main()

	assert.Equal(t, 0, gotCode)
}
