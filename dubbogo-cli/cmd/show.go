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
	"fmt"
	"log"
)

import (
	"github.com/spf13/cobra"
)

import (
	"dubbo.apache.org/dubbo-go/v3/dubbogo-cli/metadata"
	_ "dubbo.apache.org/dubbo-go/v3/dubbogo-cli/metadata/zookeeper"
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show interfaces and methods",
	Long:  ``,
	Run:   show,
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().String("r", "r", "")
	showCmd.Flags().String("h", "h", "")
}

func show(cmd *cobra.Command, _ []string) {
	registry, err := cmd.Flags().GetString("r")
	if err != nil {
		panic(err)
	}
	host, err := cmd.Flags().GetString("h")
	if err != nil {
		panic(err)
	}
	fact, ok := metadata.GetFactory(registry)
	if !ok {
		log.Print("registry not support")
		return
	}
	methodsMap, err := fact("dubbogo-cli", []string{host}).ShowChildren()
	if err != nil {
		panic(err)
	}
	for k, v := range methodsMap {
		fmt.Printf("interface: %s\n", k)
		fmt.Printf("methods: %v\n", v)
	}
}
