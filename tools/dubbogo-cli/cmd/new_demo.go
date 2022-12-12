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
)

import (
	"github.com/spf13/cobra"
)

import (
	"dubbo.apache.org/dubbo-go/v3/dubbogo-cli/generator/sample"
)

// newDemoCmd represents the new command
var newDemoCmd = &cobra.Command{
	Use:   "newDemo",
	Short: "new a dubbo-go demo project with client and server",
	Run:   createDemo,
}

func init() {
	rootCmd.AddCommand(newDemoCmd)
}

func createDemo(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Please tell me the generate path, like '.' ")
		return
	}
	path := args[0]
	if err := sample.Generate(path); err != nil {
		fmt.Printf("generate error: %s\n", err)
	}
}
