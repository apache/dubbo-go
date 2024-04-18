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
	"fmt"
)

import (
	"github.com/pkg/errors"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cmd/protoc-gen-go-dubbo/util"
	"dubbo.apache.org/dubbo-go/v3/proto/hessian2_extend"
)

var (
	ErrStreamMethod               = errors.New("dubbo doesn't support stream method")
	ErrMoreExtendArgsRespFieldNum = errors.New("extend args for response message should only has 1 field")
	ErrNoExtendArgsRespFieldNum   = errors.New("extend args for response message should has a field")
)

type DubboGo struct {
	*protogen.File

	Source       string
	ProtoPackage string
	Services     []*Service
}

type Service struct {
	ServiceName   string
	InterfaceName string
	Methods       []*Method
}

type Method struct {
	MethodName string
	InvokeName string

	// empty when RequestExtendArgs is true
	RequestType       string
	RequestExtendArgs bool
	ArgsType          []string
	ArgsName          []string

	ResponseExtendArgs bool
	ReturnType         string
}

func ProcessProtoFile(g *protogen.GeneratedFile, file *protogen.File) (*DubboGo, error) {
	desc := file.Proto
	dubboGo := &DubboGo{
		File:         file,
		Source:       desc.GetName(),
		ProtoPackage: desc.GetPackage(),
		Services:     make([]*Service, 0),
	}

	for _, service := range file.Services {
		serviceMethods := make([]*Method, 0)
		for _, method := range service.Methods {
			if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
				return nil, ErrStreamMethod
			}
			m := &Method{
				MethodName:  method.GoName,
				RequestType: g.QualifiedGoIdent(method.Input.GoIdent),
				ReturnType:  g.QualifiedGoIdent(method.Output.GoIdent),
			}

			methodOpt, ok := proto.GetExtension(method.Desc.Options(), hessian2_extend.E_MethodExtend).(*hessian2_extend.Hessian2MethodOptions)
			invokeName := util.ToLower(method.GoName)
			if ok && methodOpt != nil {
				invokeName = methodOpt.MethodName
			}
			m.InvokeName = invokeName

			inputOpt, ok := proto.GetExtension(method.Input.Desc.Options(), hessian2_extend.E_MessageExtend).(*hessian2_extend.Hessian2MessageOptions)
			if ok && inputOpt.ExtendArgs {
				m.RequestExtendArgs = true
				for _, field := range method.Input.Fields {
					// TODO(Yuukirn): auto import go_package may be caused by fieldGoType
					goType, _ := util.FieldGoType(g, field)
					m.ArgsType = append(m.ArgsType, goType)
					m.ArgsName = append(m.ArgsName, util.ToLower(field.GoName))
				}
			}

			outputOpt, ok := proto.GetExtension(method.Output.Desc.Options(), hessian2_extend.E_MessageExtend).(*hessian2_extend.Hessian2MessageOptions)
			if ok && outputOpt != nil && outputOpt.ExtendArgs {
				m.ResponseExtendArgs = true
				if len(method.Output.Fields) == 0 {
					return nil, ErrNoExtendArgsRespFieldNum
				}
				if len(method.Output.Fields) != 1 {
					return nil, ErrMoreExtendArgsRespFieldNum
				}
				goType, _ := util.FieldGoType(g, method.Output.Fields[0])
				m.ReturnType = goType
			}

			serviceMethods = append(serviceMethods, m)
		}

		serviceOpt, ok := proto.GetExtension(service.Desc.Options(), hessian2_extend.E_ServiceExtend).(*hessian2_extend.Hessian2ServiceOptions)
		interfaceName := fmt.Sprintf("%s.%s", dubboGo.ProtoPackage, service.GoName)
		if ok && serviceOpt != nil {
			interfaceName = serviceOpt.InterfaceName
		}
		dubboGo.Services = append(dubboGo.Services, &Service{
			ServiceName:   service.GoName,
			Methods:       serviceMethods,
			InterfaceName: interfaceName,
		})
	}

	return dubboGo, nil
}
