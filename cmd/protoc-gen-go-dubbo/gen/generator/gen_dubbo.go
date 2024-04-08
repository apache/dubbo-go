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

package generator

import (
	"os"
	"path/filepath"
	"strings"
)

import (
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cmd/protoc-gen-go-dubbo/util"
)

type DubboGo struct {
	Source       string
	Package      string
	Path         string
	FileName     string
	ProtoPackage string
	Services     []Service
}

type Service struct {
	ServiceName string
	Methods     []Method
}

type Method struct {
	MethodName  string
	RequestType string
	ReturnType  string
}

func ProcessProtoFile(file *descriptor.FileDescriptorProto) (DubboGo, error) {
	dubboGo := DubboGo{
		Source:       file.GetName(),
		ProtoPackage: file.GetPackage(),
		Services:     make([]Service, 0),
	}
	for _, service := range file.GetService() {
		serviceMethods := make([]Method, 0)

		for _, method := range service.GetMethod() {
			// TODO(Yuukirn): handle stream -> error
			serviceMethods = append(serviceMethods, Method{
				MethodName:  method.GetName(),
				RequestType: util.ToUpper(strings.Split(method.GetInputType(), ".")[len(strings.Split(method.GetInputType(), "."))-1]),
				ReturnType:  util.ToUpper(strings.Split(method.GetOutputType(), ".")[len(strings.Split(method.GetOutputType(), "."))-1]),
			})
		}

		dubboGo.Services = append(dubboGo.Services, Service{
			ServiceName: service.GetName(),
			Methods:     serviceMethods,
		})
	}

	pkgs := strings.Split(file.Options.GetGoPackage(), ";")
	if len(pkgs) < 2 || pkgs[1] == "" {
		return dubboGo, errors.New("need to set the package name in go_package")
	}
	dubboGo.Package = pkgs[1]
	goPkg := strings.Trim(pkgs[0], "/")
	moduleName, err := util.GetModuleName()
	if err != nil {
		return dubboGo, err
	}

	if strings.Contains(goPkg, moduleName) {
		dubboGo.Path = strings.TrimPrefix(goPkg, moduleName)
	} else {
		dubboGo.Path = goPkg
	}
	_, fileName := filepath.Split(file.GetName())
	dubboGo.FileName = strings.Split(fileName, ".")[0]
	return dubboGo, nil
}

func GenDubboFile(dubbo DubboGo) error {
	moduleDir, err := util.GetModuleDir()
	if err != nil {
		return err
	}

	goOut := filepath.Join(moduleDir, dubbo.Path, util.AddDubboGoPrefix(dubbo.FileName))
	data, err := parseDubboToString(dubbo)
	if err != nil {
		return err
	}
	return generateToFile(goOut, []byte(data))
}

func parseDubboToString(dubbo DubboGo) (string, error) {
	var builder strings.Builder

	for _, tpl := range Tpls {
		err := tpl.Execute(&builder, dubbo)
		if err != nil {
			return "", err
		}
	}

	return builder.String(), nil
}

func generateToFile(filePath string, data []byte) error {
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		return err
	}
	err = os.WriteFile(filePath, data, 0666)
	if err != nil {
		return err
	}

	return util.GoFmtFile(filePath)
}
