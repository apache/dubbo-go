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

package definition

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// ServiceDefiner is a interface of service's definition
type ServiceDefiner interface {
	ToBytes() ([]byte, error)
}

// ServiceDefinition is the describer of service definition
type ServiceDefinition struct {
	CanonicalName string
	CodeSource    string
	Methods       []MethodDefinition
	Types         []TypeDefinition
}

// ToBytes convert ServiceDefinition to json string
func (def *ServiceDefinition) ToBytes() ([]byte, error) {
	return json.Marshal(def)
}

// String will iterate all methods and parameters and convert them to json string
func (def *ServiceDefinition) String() string {
	var methodStr strings.Builder
	for _, m := range def.Methods {
		var paramType strings.Builder
		for _, p := range m.ParameterTypes {
			paramType.WriteString(fmt.Sprintf("{type:%v}", p))
		}
		var param strings.Builder
		for _, d := range m.Parameters {
			param.WriteString(fmt.Sprintf("{id:%v,type:%v,builderName:%v}", d.ID, d.Type, d.TypeBuilderName))
		}
		methodStr.WriteString(fmt.Sprintf("{name:%v,parameterTypes:[%v],returnType:%v,params:[%v] }", m.Name, paramType.String(), m.ReturnType, param.String()))
	}
	var types strings.Builder
	for _, d := range def.Types {
		types.WriteString(fmt.Sprintf("{id:%v,type:%v,builderName:%v}", d.ID, d.Type, d.TypeBuilderName))
	}
	return fmt.Sprintf("{canonicalName:%v, codeSource:%v, methods:[%v], types:[%v]}", def.CanonicalName, def.CodeSource, methodStr.String(), types.String())
}

// FullServiceDefinition is the describer of service definition with parameters
type FullServiceDefinition struct {
	Parameters map[string]string
	ServiceDefinition
}

// MethodDefinition is the describer of method definition
type MethodDefinition struct {
	Name           string
	ParameterTypes []string
	ReturnType     string
	Parameters     []TypeDefinition
}

// TypeDefinition is the describer of type definition
type TypeDefinition struct {
	ID              string
	Type            string
	Items           []TypeDefinition
	Enums           []string
	Properties      map[string]TypeDefinition
	TypeBuilderName string
}

// BuildServiceDefinition can build service definition which will be used to describe a service
func BuildServiceDefinition(service common.Service, url *common.URL) *ServiceDefinition {
	sd := &ServiceDefinition{}
	sd.CanonicalName = url.Service()

	for k, m := range service.Method() {
		var paramTypes []string
		if len(m.ArgsType()) > 0 {
			for _, t := range m.ArgsType() {
				paramTypes = append(paramTypes, t.Kind().String())
			}
		}

		var returnType string
		if m.ReplyType() != nil {
			returnType = m.ReplyType().Kind().String()
		}

		methodD := MethodDefinition{
			Name:           k,
			ParameterTypes: paramTypes,
			ReturnType:     returnType,
		}
		sd.Methods = append(sd.Methods, methodD)
	}

	return sd
}

// BuildFullDefinition can build service definition with full url parameters
func BuildFullDefinition(service common.Service, url *common.URL) *FullServiceDefinition {
	fsd := &FullServiceDefinition{}
	sd := BuildServiceDefinition(service, url)
	fsd.ServiceDefinition = *sd
	fsd.Parameters = make(map[string]string)
	for k, v := range url.GetParams() {
		fsd.Parameters[k] = strings.Join(v, ",")
	}
	return fsd
}

// ToBytes convert ServiceDefinition to json string
func (def *FullServiceDefinition) ToBytes() ([]byte, error) {
	return json.Marshal(def)
}

// String will iterate all methods and parameters and convert them to json string
func (def *FullServiceDefinition) String() string {
	var methodStr strings.Builder
	for _, m := range def.Methods {
		var paramType strings.Builder
		for _, p := range m.ParameterTypes {
			paramType.WriteString(fmt.Sprintf("{type:%v}", p))
		}
		var param strings.Builder
		for _, d := range m.Parameters {
			param.WriteString(fmt.Sprintf("{id:%v,type:%v,builderName:%v}", d.ID, d.Type, d.TypeBuilderName))
		}
		methodStr.WriteString(fmt.Sprintf("{name:%v,parameterTypes:[%v],returnType:%v,params:[%v] }", m.Name, paramType.String(), m.ReturnType, param.String()))
	}
	var types strings.Builder
	for _, d := range def.Types {
		types.WriteString(fmt.Sprintf("{id:%v,type:%v,builderName:%v}", d.ID, d.Type, d.TypeBuilderName))
	}

	sortSlice := make([]string, 0)
	var parameters strings.Builder
	for k := range def.Parameters {
		sortSlice = append(sortSlice, k)
	}
	sort.Slice(sortSlice, func(i, j int) bool { return sortSlice[i] < sortSlice[j] })
	for _, k := range sortSlice {
		parameters.WriteString(fmt.Sprintf("%v:%v,", k, def.Parameters[k]))
	}

	return fmt.Sprintf("{parameters:{%v}, canonicalName:%v, codeSource:%v, methods:[%v], types:[%v]}",
		strings.TrimRight(parameters.String(), ","), def.CanonicalName, def.CodeSource, methodStr.String(), types.String())
}

// ServiceDescriperBuild builds the service key, format is `group/serviceName:version` which be same as URL's service key
func ServiceDescriperBuild(serviceName string, group string, version string) string {
	buf := &bytes.Buffer{}
	if group != "" {
		buf.WriteString(group)
		buf.WriteString(constant.PathSeparator)
	}
	buf.WriteString(serviceName)
	if version != "" && version != "0.0.0" {
		buf.WriteString(constant.KeySeparator)
		buf.WriteString(version)
	}
	return buf.String()
}
