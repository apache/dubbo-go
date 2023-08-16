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
	bastPath, err := os.Getwd()
	if err != nil {
		return err
	}
	triple.Source, err = filepath.Rel(bastPath, g.ctx.Src)
	if err != nil {
		return err
	}
	data, err := g.parseTripleToString(triple)
	if err != nil {
		return err
	}
	g.parseGOOut(triple)
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
			tripleGo.Package = p.Name + "triple"
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

func (g *Generator) parseGOOut(triple TripleGo) {
	prefix := strings.TrimPrefix(triple.Import, g.ctx.GoModuleName)
	g.ctx.GoOut = filepath.Join(g.ctx.Pwd, filepath.Join(prefix, triple.Package+"/"+triple.ProtoPackage+".triple.go"))
}

func (g *Generator) generateToFile(filePath string, data []byte) error {
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0666)
}

type TripleGo struct {
	Source       string
	Package      string
	Import       string
	ProtoPackage string
	Services     []Service
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
