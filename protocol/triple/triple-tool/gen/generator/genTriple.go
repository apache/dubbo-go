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
	"github.com/emicklei/proto"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

import (
	"dubbo.apache.org/dubbo-go/v3/triple-tool/util"
)

func (g *Generator) GenTriple() error {
	p, err := g.parseFileToProto(g.ctx.Src)
	if err != nil {
		return err
	}
	triple, err := g.parseProtoToTriple(p)
	if err != nil {
		return err
	}
	basePath, err := os.Getwd()
	if err != nil {
		return err
	}
	triple.Source, err = filepath.Rel(basePath, g.ctx.Src)
	if err != nil {
		return err
	}
	data, err := g.parseTripleToString(triple)
	if err != nil {
		return err
	}
	g.parseGOout(triple)
	return g.generateToFile(g.ctx.GoOut, []byte(data))
}

func (g *Generator) parseFileToProto(filePath string) (*proto.Proto, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	parser := proto.NewParser(file)
	p, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (g *Generator) parseProtoToTriple(p *proto.Proto) (TripleGo, error) {
	var tripleGo TripleGo
	proto.Walk(
		p,
		proto.WithPackage(func(p *proto.Package) {
			tripleGo.ProtoPackage = p.Name
			tripleGo.Package = p.Name
		}),
		proto.WithService(func(p *proto.Service) {
			s := Service{ServiceName: p.Name}
			for _, visitee := range p.Elements {
				if vi, ok := visitee.(*proto.RPC); ok {
					md := Method{
						MethodName:     vi.Name,
						RequestType:    vi.RequestType,
						StreamsRequest: vi.StreamsRequest,
						ReturnType:     vi.ReturnsType,
						StreamsReturn:  vi.StreamsReturns,
					}
					s.Methods = append(s.Methods, md)
					if md.StreamsRequest || md.StreamsReturn {
						tripleGo.IsStream = true
					}
				}
			}
			tripleGo.Services = append(tripleGo.Services, s)
		}),
		proto.WithOption(func(p *proto.Option) {
			if p.Name == "go_package" {
				i := p.Constant.Source
				i = strings.Trim(i, "/")
				if strings.Contains(i, g.ctx.GoModuleName) {
					tripleGo.Import = strings.Split(i, ";")[0]
				} else {
					tripleGo.Import = g.ctx.GoModuleName + "/" + strings.Split(i, ";")[0]
				}
			}
		}),
	)
	return tripleGo, nil
}

func (g *Generator) parseTripleToString(t TripleGo) (string, error) {
	var builder strings.Builder

	for _, tpl := range Tpls {
		err := tpl.Execute(&builder, t)
		if err != nil {
			return "", err
		}
	}

	return builder.String(), nil
}

func (g *Generator) parseGOout(triple TripleGo) {
	prefix := strings.TrimPrefix(triple.Import, g.ctx.GoModuleName)
	g.ctx.GoOut = filepath.Join(g.ctx.ModuleDir, filepath.Join(prefix, triple.Package+"triple/"+triple.ProtoPackage+".triple.go"))
}

func (g *Generator) generateToFile(filePath string, data []byte) error {
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

func ProcessProtoFile(file *descriptor.FileDescriptorProto) (TripleGo, error) {
	tripleGo := TripleGo{
		Source:       file.GetName(),
		Package:      file.GetPackage(),
		ProtoPackage: file.GetPackage(),
		Services:     make([]Service, 0),
	}
	for _, service := range file.GetService() {
		serviceMethods := make([]Method, 0)

		for _, method := range service.GetMethod() {
			serviceMethods = append(serviceMethods, Method{
				MethodName:     method.GetName(),
				RequestType:    strings.Split(method.GetInputType(), ".")[len(strings.Split(method.GetInputType(), "."))-1],
				StreamsRequest: method.GetClientStreaming(),
				ReturnType:     strings.Split(method.GetOutputType(), ".")[len(strings.Split(method.GetOutputType(), "."))-1],
				StreamsReturn:  method.GetServerStreaming(),
			})
			if method.GetClientStreaming() || method.GetServerStreaming() {
				tripleGo.IsStream = true
			}
		}

		tripleGo.Services = append(tripleGo.Services, Service{
			ServiceName: service.GetName(),
			Methods:     serviceMethods,
		})
	}

	goPkg := file.Options.GetGoPackage()
	goPkg = strings.Split(goPkg, ";")[0]
	goPkg = strings.Trim(goPkg, "/")
	moduleName, err := util.GetModuleName()
	if err != nil {
		return tripleGo, err
	}

	if strings.Contains(goPkg, moduleName) {
		tripleGo.Import = strings.Split(goPkg, ";")[0]
	} else {
		tripleGo.Import = moduleName + "/" + strings.Split(goPkg, ";")[0]
	}

	return tripleGo, nil
}

func GenTripleFile(triple TripleGo) error {
	module, err := util.GetModuleName()
	if err != nil {
		return err
	}
	prefix := strings.TrimPrefix(triple.Import, module)
	moduleDir, err := util.GetModuleDir()
	if err != nil {
		return err
	}
	GoOut := filepath.Join(moduleDir, filepath.Join(prefix, triple.Package+"triple/"+triple.ProtoPackage+".triple.go"))
	g := &Generator{}
	data, err := g.parseTripleToString(triple)
	if err != nil {
		return err
	}
	return g.generateToFile(GoOut, []byte(data))
}

type TripleGo struct {
	Source       string
	Package      string
	Import       string
	ProtoPackage string
	Services     []Service
	IsStream     bool
}

type Service struct {
	ServiceName string
	Methods     []Method
}

type Method struct {
	MethodName     string
	RequestType    string
	StreamsRequest bool
	ReturnType     string
	StreamsReturn  bool
}
