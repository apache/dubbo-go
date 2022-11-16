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
	showCmd.Flags().String("r", "", "")
	showCmd.Flags().String("mc", "", "Get Metadata in MetadataCenter")
	showCmd.Flags().String("h", "h", "")

}

func show(cmd *cobra.Command, _ []string) {
	var (
		methodsMap map[string][]string
		err        error
	)

	registry, err := cmd.Flags().GetString("r")
	if err != nil {
		panic(err)
	}

	metadataCenter, err := cmd.Flags().GetString("mc")
	if err != nil {
		panic(err)
	}

	host, err := cmd.Flags().GetString("h")
	if err != nil {
		panic(err)
	}

	if registry != "" {
		registryFactory, ok := metadata.GetFactory(registry)
		if !ok {
			log.Print("registry not support")
			return
		}
		methodsMap, err = registryFactory("dubbogo-cli", []string{host}).ShowRegistryCenterChildren()
		if err != nil {
			panic(err)
		}
		fmt.Printf("======================\n")
		fmt.Printf("Registry:\n")
		for k, v := range methodsMap {
			fmt.Printf("interface: %s\n", k)
			fmt.Printf("methods: %v\n", v)
		}
		fmt.Printf("======================\n")
	}

	if metadataCenter != "" {
		metadataCenterFactory, ok := metadata.GetFactory(metadataCenter)
		if !ok {
			log.Print("metadataCenter not support")
			return
		}
		methodsMap, err = metadataCenterFactory("dubbogo-cli", []string{host}).ShowMetadataCenterChildren()
		if err != nil {
			panic(err)
		}
		fmt.Printf("======================\n")
		fmt.Printf("MetadataCenter:\n")
		for k, v := range methodsMap {
			fmt.Printf("interface: %s\n", k)
			fmt.Printf("methods: %v\n", v)
		}
		fmt.Printf("======================\n")
	}
}
