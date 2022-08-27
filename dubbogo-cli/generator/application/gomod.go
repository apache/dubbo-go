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

package application

const (
	gomodFile = `module dubbo-go-app

go 1.17

require (
	dubbo.apache.org/dubbo-go/v3 v3.0.1
	github.com/dubbogo/grpc-go v1.42.9
	github.com/dubbogo/triple v1.1.8
	github.com/golang/protobuf v1.5.2
	google.golang.org/protobuf v1.27.1
)
`
)

func init() {
	fileMap["gomodFile"] = &fileGenerator{
		path:    ".",
		file:    "go.mod",
		context: gomodFile,
	}
}
