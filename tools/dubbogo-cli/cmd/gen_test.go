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
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/dubbogo-cli/generator/application"
	"dubbo.apache.org/dubbo-go/v3/dubbogo-cli/generator/sample"
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
	tempFiles, err := walkDir(templatePath)
	assert.Nil(t, err)
	for _, tempPath := range tempFiles {
		newGenetedPath := strings.ReplaceAll(tempPath, "/template/", "/")
		newGenetedFile, err := os.ReadFile(newGenetedPath)
		assert.Nil(t, err)
		tempFile, err := os.ReadFile(tempPath)
		assert.Nil(t, err)
		assert.Equal(t, string(tempFile), string(newGenetedFile))
	}
	os.RemoveAll(genPath)
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
