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

package cmd

import (
	"log"
)

import (
	"github.com/spf13/cobra"
)

import (
	"dubbo.apache.org/dubbo-go/v3/dubbogo-cli/internal/client"
	"dubbo.apache.org/dubbo-go/v3/dubbogo-cli/internal/json_register"
)

// callCmd represents the call command
var (
	callCmd = &cobra.Command{
		Use:   "call",
		Short: "call a method",
		Long:  "",
		Run:   call,
	}
)

var (
	host            string
	port            int
	protocolName    string
	InterfaceID     string
	version         string
	group           string
	method          string
	sendObjFilePath string
	recvObjFilePath string
	timeout         int
)

func init() {
	rootCmd.AddCommand(callCmd)

	callCmd.Flags().StringVarP(&host, "h", "", "localhost", "target server host")
	callCmd.Flags().IntVarP(&port, "p", "", 8080, "target server port")
	callCmd.Flags().StringVarP(&protocolName, "proto", "", "dubbo", "transfer protocol")
	callCmd.Flags().StringVarP(&InterfaceID, "i", "", "com", "target service registered interface")
	callCmd.Flags().StringVarP(&version, "v", "", "", "target service version")
	callCmd.Flags().StringVarP(&group, "g", "", "", "target service group")
	callCmd.Flags().StringVarP(&method, "method", "", "", "target method")
	callCmd.Flags().StringVarP(&sendObjFilePath, "sendObj", "", "", "json file path to define transfer struct")
	callCmd.Flags().StringVarP(&recvObjFilePath, "recvObj", "", "", "json file path to define receive struct")
	callCmd.Flags().IntVarP(&timeout, "timeout", "", 3000, "request timeout (ms)")
}

func call(_ *cobra.Command, _ []string) {
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
