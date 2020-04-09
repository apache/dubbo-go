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
	"github.com/apache/dubbo-go/common"
)

// ServiceDefinition is the describer of service definition
type ServiceDefinition struct {
	CanonicalName string
	CodeSource    string
	Methods       []MethodDefinition
	Types         []TypeDefinition
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
	Id              string
	Type            string
	Items           []TypeDefinition
	Enums           []string
	Properties      map[string]TypeDefinition
	TypeBuilderName string
}

// BuildServiceDefinition can build service definition which will be used to describe a service
func BuildServiceDefinition(service common.Service, url common.URL) ServiceDefinition {
	sd := ServiceDefinition{}
	sd.CanonicalName = url.Service()

	for k, m := range service.Method() {
		var paramTypes []string
		for _, t := range m.ArgsType() {
			paramTypes = append(paramTypes, t.Kind().String())
		}
		methodD := MethodDefinition{
			Name:           k,
			ParameterTypes: paramTypes,
			ReturnType:     m.ReplyType().Kind().String(),
		}
		sd.Methods = append(sd.Methods, methodD)
	}

	return sd
}

// ServiceDescriperBuild: build the service key, format is `group/serviceName:version` which be same as URL's service key
func ServiceDescriperBuild(serviceName string, group string, version string) string {
	buf := &bytes.Buffer{}
	if group != "" {
		buf.WriteString(group)
		buf.WriteString("/")
	}
	buf.WriteString(serviceName)
	if version != "" && version != "0.0.0" {
		buf.WriteString(":")
		buf.WriteString(version)
	}
	return buf.String()
}
