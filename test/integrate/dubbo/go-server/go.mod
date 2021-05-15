module dubbo.apache.org/dubbo-go/v3/test/integrate/dubbo/go-server

go 1.13

require (
	dubbo.apache.org/dubbo-go/v3 v3.0.0
	// useless dubbo-go version number, it will be replaced when run by this Docker file: ../Dockerfile:
	//```
	// RUN test ${PR_ORIGIN_REPO} && go mod edit -replace=dubbo.apache.org/dubbo-go/v3=github.com/${PR_ORIGIN_REPO}@${PR_ORIGIN_COMMITID} || go get -u dubbo.apache.org/dubbo-go/v3@develop
	//```
	github.com/apache/dubbo-go-hessian2 v1.9.1
)

replace dubbo.apache.org/dubbo-go/v3 => ../../../../../dubbo-go
