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

package json_register

import (
	"fmt"
	"log"
	"reflect"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	jparser "github.com/dubbogo/gost/encoding/json"
)

import (
	"github.com/apache/dubbo-go/tools/cli/common"
)

// RegisterStructFromFile create the interface defined by @path file, and register it to hessian
// the interface defined must have field "JavaClassName"
func RegisterStructFromFile(path string) interface{} {
	if path == "" {
		return nil
	}
	pair, pkg, err := jparser.File2Interface(path)
	log.Printf("Created pkg: \n")
	common.PrintInterface(pkg)
	if err != nil {
		fmt.Println("error: json file parse failed :", err)
		return nil
	}
	for _, v := range pair {
		hessian.RegisterPOJOMapping(v.JavaClassName, v.Obj)
	}
	hessian.RegisterPOJOMapping(getJavaClassName(pkg), pkg)
	return pkg
}

func getJavaClassName(pkg interface{}) string {
	val := reflect.ValueOf(pkg).Elem()
	typ := reflect.TypeOf(pkg).Elem()
	nums := val.NumField()
	for i := 0; i < nums; i++ {
		if typ.Field(i).Name == "JavaClassName" {
			return val.Field(i).String()
		}
	}
	fmt.Println("error: JavaClassName not found")
	return ""
}
