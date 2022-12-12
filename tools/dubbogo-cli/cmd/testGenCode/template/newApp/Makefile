IMAGE = $(your_repo)/$(namespace)/$(image_name)
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
