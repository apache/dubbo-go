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

package main

import (
	"flag"
	"log"
)

import (
	"github.com/apache/dubbo-go/tools/cli/client"
	"github.com/apache/dubbo-go/tools/cli/json_register"
)

var host string
var port int
var protocolName string
var InterfaceID string
var version string
var group string
var method string
var sendObjFilePath string
var recvObjFilePath string
var timeout int

func init() {
	flag.StringVar(&host, "h", "localhost", "target server host")
	flag.IntVar(&port, "p", 8080, "target server port")
	flag.StringVar(&protocolName, "proto", "dubbo", "transfer protocol")
	flag.StringVar(&InterfaceID, "i", "com", "target service registered interface")
	flag.StringVar(&version, "v", "", "target service version")
	flag.StringVar(&group, "g", "", "target service group")
	flag.StringVar(&method, "method", "", "target method")
	flag.StringVar(&sendObjFilePath, "sendObj", "", "json file path to define transfer struct")
	flag.StringVar(&recvObjFilePath, "recvObj", "", "json file path to define receive struct")
	flag.IntVar(&timeout, "timeout", 3000, "request timeout (ms)")
}

func checkParam() {
	if method == "" {
		log.Fatalln("-method value not fond")
	}
	if sendObjFilePath == "" {
		log.Fatalln("-sendObj value not found")
	}
	if recvObjFilePath == "" {
		log.Fatalln("-recObj value not found")
	}
}

func main() {
	flag.Parse()
	checkParam()
	reqPkg := json_register.RegisterStructFromFile(sendObjFilePath)
	recvPkg := json_register.RegisterStructFromFile(recvObjFilePath)

	t, err := client.NewTelnetClient(host, port, protocolName, InterfaceID, version, group, method, reqPkg, timeout)
	if err != nil {
		panic(err)
	}
	t.ProcessRequests(recvPkg)
	t.Destroy()
}
