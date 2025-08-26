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

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/tools/dubbogo-cli/generator/application"
	"dubbo.apache.org/dubbo-go/v3/tools/dubbogo-cli/generator/sample"
)

func TestNewApp(t *testing.T) {
	if err := application.Generate("./testGenCode/newApp"); err != nil {
		fmt.Printf("generate error: %s\n", err)
	}

	assertFileSame(t, "./testGenCode/newApp", "./testGenCode/template/newApp")
}

func TestNewDemo(t *testing.T) {
	if err := sample.Generate("./testGenCode/newDemo"); err != nil {
		fmt.Printf("generate error: %s\n", err)
	}

	assertFileSame(t, "./testGenCode/newDemo", "./testGenCode/template/newDemo")
}

func assertFileSame(t *testing.T, genPath, templatePath string) {
	t.Cleanup(func() {
		os.RemoveAll(genPath)
	})

	// 1. get all files in template directory
	templateFiles, err := walkDir(templatePath)
	require.NoError(t, err, "iterate template directory failed")

	// 2. get all files in generated directory
	genFiles, err := walkDir(genPath)
	require.NoError(t, err, "iterate generated directory failed")

	// 3. assert the number of generated files matches the template files
	require.Len(t, genFiles, len(templateFiles), "number of generated files does not match template files")

	// 4. compare each file in the template directory with the corresponding file in the generated directory
	for _, tempFileAbsPath := range templateFiles {
		// get relative paths of the template files
		relPath, err := filepath.Rel(templatePath, tempFileAbsPath)
		require.NoError(t, err)

		genFileAbsPath := filepath.Join(genPath, relPath)

		require.FileExists(t, genFileAbsPath, "file not exist: %s", genFileAbsPath)

		tempContent, err := os.ReadFile(tempFileAbsPath)
		require.NoError(t, err)
		normalizedTempContent := strings.ReplaceAll(string(tempContent), "\r\n", "\n")

		genContent, err := os.ReadFile(genFileAbsPath)
		normalizedGenContent := strings.ReplaceAll(string(genContent), "\r\n", "\n")
		require.NoError(t, err)

		require.Equal(t, normalizedTempContent, normalizedGenContent, "get different file content: %s", genFileAbsPath)
	}
}

func walkDir(dirPth string) (files []string, err error) {
	files = make([]string, 0, 30)

	err = filepath.Walk(dirPth, func(filename string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		files = append(files, filename)
		return nil
	})

	return files, err
}
