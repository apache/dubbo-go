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
	"os/exec"
)

import (
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install tools of dubbo-go ecology.",
	Run:   InstallCommand,
}

type InstallFactory interface {
	GetCmdName() string
	GetPackage() string
}

type InstallFormatter struct {
}

func (InstallFormatter) GetCmdName() string {
	return "formatter"
}
func (InstallFormatter) GetPackage() string {
	return "github.com/dubbogo/tools/cmd/imports-formatter@latest"
}

type InstallDubbo3Grpc struct {
}

func (InstallDubbo3Grpc) GetCmdName() string {
	return "dubbo3grpc"
}
func (InstallDubbo3Grpc) GetPackage() string {
	return "github.com/dubbogo/tools/cmd/protoc-gen-dubbo3grpc@latest"
}

type Installtriple struct {
}

func (Installtriple) GetCmdName() string {
	return "triple"
}
func (Installtriple) GetPackage() string {
	return "github.com/dubbogo/tools/cmd/protoc-gen-go-triple@v1.0.10-rc2"
}

var installFactory = make(map[string]InstallFactory)

func registerInstallFactory(f InstallFactory) {
	installFactory[f.GetCmdName()] = f
}

func init() {
	rootCmd.AddCommand(installCmd)
	registerInstallFactory(&InstallFormatter{})
	registerInstallFactory(&InstallDubbo3Grpc{})
	registerInstallFactory(&Installtriple{})
}

func InstallCommand(cmd *cobra.Command, args []string) {
	argFilter := make(map[string]InstallFactory)

	var f InstallFactory
	var existed bool
	for _, arg := range args {
		fName := arg
		if f, existed = installFactory[fName]; !existed {
			f = nil
		}
		argFilter[arg] = f
	}

	if _, existed = argFilter["all"]; existed {
		delete(argFilter, "all")
		for k, f := range installFactory {
			argFilter[k] = f
		}
	}

	var k string
	for k, f = range argFilter {
		if f != nil {
			fmt.Println("go", "install", f.GetPackage())
			cmd := exec.Command("go", "install", f.GetPackage())
			if _, err := cmd.StdoutPipe(); err != nil { //获取输出对象，可以从该对象中读取输出结果
				fmt.Println(err)
			} else {
				if err := cmd.Run(); err != nil { // 运行命令
					fmt.Println(err)
				}
			}
			continue
		}
		fmt.Println("不支持安装 " + k)
	}

}
