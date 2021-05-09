module dubbo.apache.org/dubbogo/test/integrate/dubbo/go-server/v3

go 1.15

require (
	dubbo.apache.org/dubbogo/v3 v3.0.0-00010101000000-000000000000
	// useless dubbo-go version number, it will be replaced when run by this Docker file: ../Dockerfile:
	//```
	// RUN test ${PR_ORIGIN_REPO} && go mod edit  -replace=github.com/apache/dubbo-go=github.com/${PR_ORIGIN_REPO}@${PR_ORIGIN_COMMITID} || go get -u github.com/apache/dubbo-go@develop
	//```
	github.com/apache/dubbo-go-hessian2 v1.9.1
)

//replace dubbo.apache.org/dubbogo/v3 => ../../../../../dubbo-go
