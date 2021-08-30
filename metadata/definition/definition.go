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
	"reflect"
	"strings"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

// ServiceDefiner is a interface of service's definition
type ServiceDefiner interface {
	ToBytes() ([]byte, error)
}

// ServiceDefinition is the describer of service definition
type ServiceDefinition struct {
	CanonicalName string             `json:"canonicalName"`
	CodeSource    string             `json:"codeSource"`
	Methods       []MethodDefinition `json:"methods"`
	Types         []TypeDefinition   `json:"types"`
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
			param.WriteString(fmt.Sprintf("{id:%v,type:%v,builderName:%v}", d.Id, d.Type, d.TypeBuilderName))
		}
		methodStr.WriteString(fmt.Sprintf("{name:%v,parameterTypes:[%v],returnType:%v,params:[%v] }", m.Name, paramType.String(), m.ReturnType, param.String()))
	}
	var types strings.Builder
	for _, d := range def.Types {
		types.WriteString(fmt.Sprintf("{id:%v,type:%v,builderName:%v}", d.Id, d.Type, d.TypeBuilderName))
	}
	return fmt.Sprintf("{canonicalName:%v, codeSource:%v, methods:[%v], types:[%v]}", def.CanonicalName, def.CodeSource, methodStr.String(), types.String())
}

// FullServiceDefinition is the describer of service definition with parameters
type FullServiceDefinition struct {
	ServiceDefinition
	Params map[string]string
}

// MethodDefinition is the describer of method definition
type MethodDefinition struct {
	Name           string           `json:"name"`
	ParameterTypes []string         `json:"parameterTypes"`
	ReturnType     string           `json:"returnType"`
	Parameters     []TypeDefinition `json:"parameters"`
}

// TypeDefinition is the describer of type definition
type TypeDefinition struct {
	Id              string                    `json:"id"`
	Type            string                    `json:"type"`
	Items           []TypeDefinition          `json:"items"`
	Enums           []string                  `json:"enums"`
	Properties      map[string]TypeDefinition `json:"properties"`
	TypeBuilderName string                    `json:"typeBuilderName"`
}

// BuildServiceDefinition can build service definition which will be used to describe a service
func BuildServiceDefinition(service common.Service, url *common.URL) *ServiceDefinition {
	sd := &ServiceDefinition{}
	sd.CanonicalName = url.Service()

	for k, m := range service.Method() {
		var paramTypes []string
		var param string
		if len(m.ArgsType()) > 0 {
			for _, t := range m.ArgsType() {
				if t.Kind() == reflect.Ptr {
					param = getArgType(reflect.New(t).Interface())
				} else {
					param = t.Kind().String()
				}
				paramTypes = append(paramTypes, param)
			}
		}

		var returnType string

		if m.ReplyType() != nil {
			if m.ReplyType().Kind() == reflect.Ptr {
				returnType = getArgType(reflect.New(m.ReplyType()).Interface())
			} else {
				returnType = m.ReplyType().Kind().String()
			}
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

// ServiceDescriperBuild: build the service key, format is `group/serviceName:version` which be same as URL's service key
func ServiceDescriperBuild(serviceName string, group string, version string) string {
	buf := &bytes.Buffer{}
	if group != "" {
		buf.WriteString(group)
		buf.WriteString(constant.PATH_SEPARATOR)
	}
	buf.WriteString(serviceName)
	if version != "" && version != "0.0.0" {
		buf.WriteString(constant.KEY_SEPARATOR)
		buf.WriteString(version)
	}
	return buf.String()
}

func getArgType(v interface{}) string {
	if v == nil {
		return "V"
	}

	v = reflect.ValueOf(v).Elem().Interface()

	switch v.(type) {
	// Serialized tags for base types
	case nil:
		return "V"
	case bool:
		return "Z"
	case []bool:
		return "[Z"
	case byte:
		return "B"
	case []byte:
		return "[B"
	case int8:
		return "B"
	case []int8:
		return "[B"
	case int16:
		return "S"
	case []int16:
		return "[S"
	case uint16: // Equivalent to Char of Java
		return "C"
	case []uint16:
		return "[C"
	// case rune:
	//	return "C"
	case int:
		return "J"
	case []int:
		return "[J"
	case int32:
		return "I"
	case []int32:
		return "[I"
	case int64:
		return "J"
	case []int64:
		return "[J"
	case time.Time:
		return "java.util.Date"
	case []time.Time:
		return "[Ljava.util.Date"
	case float32:
		return "F"
	case []float32:
		return "[F"
	case float64:
		return "D"
	case []float64:
		return "[D"
	case string:
		return "java.lang.String"
	case []string:
		return "[Ljava.lang.String;"
	case []hessian.Object:
		return "[Ljava.lang.Object;"
	case map[interface{}]interface{}:
		// return  "java.util.HashMap"
		return "java.util.Map"
	case hessian.POJO:
		return v.(hessian.POJO).JavaClassName()
	//  Serialized tags for complex types
	default:
		t := reflect.TypeOf(v)
		if reflect.Ptr == t.Kind() {
			t = reflect.TypeOf(reflect.ValueOf(v).Elem())
		}
		switch t.Kind() {
		case reflect.Struct:
			v, ok := v.(hessian.POJO)
			if ok {
				return v.JavaClassName()
			}
			return "java.lang.Object"
		case reflect.Slice, reflect.Array:
			if t.Elem().Kind() == reflect.Struct {
				return "[Ljava.lang.Object;"
			}
			// return "java.util.ArrayList"
			return "java.util.List"
		case reflect.Map: // Enter here, map may be map[string]int
			return "java.util.Map"
		default:
			return ""
		}
	}

	// unreachable
	// return "java.lang.RuntimeException"
}
