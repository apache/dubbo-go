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
