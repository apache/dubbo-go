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
	makefile = `IMAGE = $(your_repo)/$(namespace)/$(image_name)
TAG = 1.0.0
HELM_INSTALL_NAME = dubbo-go-app

build-amd64-app:
	GOOS=linux GOARCH=amd64 go build -o build/app ./cmd

build: proto-gen tidy build-amd64-app
	cp ./conf/dubbogo.yaml ./build/dubbogo.yaml
	docker build ./build -t ${IMAGE}:${TAG}
	docker push ${IMAGE}:${TAG}
	make clean

buildx-publish: proto-gen tidy build-amd64-app
	cp ./conf/dubbogo.yaml ./build/dubbogo.yaml
	docker buildx build \
    	 --platform linux/amd64 \
    	 -t ${IMAGE}:${TAG} \
    	 ./build --push
	make clean

remove:
	helm uninstall ${HELM_INSTALL_NAME}

deploy:
	helm install ${HELM_INSTALL_NAME} ./chart/app

deploy-nacos-env:
	helm install nacos ./chart/nacos_env

remove-nacos-env:
	helm uninstall nacos

proto-gen:
	protoc --go_out=./api --go-triple_out=./api ./api/api.proto

clean:
	rm ./build/dubbogo.yaml
	rm ./build/app

tidy:
	go mod tidy

test:
	go test ./...
`
)

func init() {
	fileMap["makefile"] = &fileGenerator{
		path:    ".",
		file:    "Makefile",
		context: makefile,
	}
}
