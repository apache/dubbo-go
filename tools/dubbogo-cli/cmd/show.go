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
	"os"
	"strings"
)

import (
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

import (
	"dubbo.apache.org/dubbo-go/v3/tools/dubbogo-cli/metadata"
	_ "dubbo.apache.org/dubbo-go/v3/tools/dubbogo-cli/metadata/zookeeper"
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show interfaces and methods from registry or metadata center",
	Long:  `Displays available interfaces and their methods from the specified registry or metadata center.`,
	Run:   show,
}

func init() {
	rootCmd.AddCommand(showCmd)
	showCmd.Flags().String("r", "", "Registry type (e.g., zookeeper)")
	showCmd.Flags().String("mc", "", "Metadata center type (e.g., zookeeper)")
	showCmd.Flags().String("h", "", "Host address of registry or metadata center (e.g., 127.0.0.1:2181)")
}

func show(cmd *cobra.Command, _ []string) {
	var (
		methodsMap map[string][]string
		err        error
	)

	registry, _ := cmd.Flags().GetString("r")
	metadataCenter, _ := cmd.Flags().GetString("mc")
	host, _ := cmd.Flags().GetString("h")

	// Validate inputs
	if registry == "" && metadataCenter == "" {
		log.Println("Error: At least one of --r (registry) or --mc (metadata center) must be specified")
		return
	}
	if host == "" {
		log.Println("Error: Host (--h) is required")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Interface", "Method"})

	if registry != "" {
		registryFactory, ok := metadata.GetFactory(registry)
		if !ok {
			log.Printf("Error: Registry type '%s' is not supported", registry)
			return
		}
		// Pass the raw host address instead of constructing a URL
		hostArg := []string{host}
		methodsMap, err = registryFactory("dubbogo-cli", hostArg).ShowRegistryCenterChildren()
		if err != nil {
			log.Printf("Failed to fetch registry data from %s: %v", registry, err)
			fmt.Printf("======================\n")
			fmt.Printf("Registry (%s): No data available\n", registry)
			fmt.Printf("======================\n")
			return
		}
		fmt.Printf("======================\n")
		fmt.Printf("Registry (%s):\n", registry)
		if len(methodsMap) == 0 {
			fmt.Println("No interfaces found")
		} else {
			for k, v := range methodsMap {
				table.Append([]string{k, strings.Join(v, ", ")})
			}
			table.Render()
		}
		fmt.Printf("======================\n")
	}

	if metadataCenter != "" {
		metadataCenterFactory, ok := metadata.GetFactory(metadataCenter)
		if !ok {
			log.Printf("Error: Metadata center type '%s' is not supported", metadataCenter)
			return
		}
		hostArg := []string{host}
		methodsMap, err = metadataCenterFactory("dubbogo-cli", hostArg).ShowMetadataCenterChildren()
		if err != nil {
			log.Printf("Error fetching metadata center data: %v", err)
			return
		}
		fmt.Printf("======================\n")
		fmt.Printf("MetadataCenter (%s):\n", metadataCenter)
		for k, v := range methodsMap {
			table.Append([]string{k, strings.Join(v, ", ")})
		}
		table.Render()
		fmt.Printf("======================\n")
	}
}
