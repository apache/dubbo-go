module github.com/apache/dubbo-go/test/integrate/dubbo/go-server

go 1.13

require (
	// useless dubbo-go version number, it will be replaced when run by this Docker file: ../Dockerfile:
	//```
	// RUN test ${PR_ORIGIN_REPO} && go mod edit  -replace=github.com/apache/dubbo-go=github.com/${PR_ORIGIN_REPO}@${PR_ORIGIN_COMMITID} || go get -u github.com/apache/dubbo-go@develop
	//```
	github.com/apache/dubbo-go v1.5.5
	github.com/apache/dubbo-go-hessian2 v1.9.1
)

replace github.com/apache/dubbo-go => ../../../../../dubbo-go
