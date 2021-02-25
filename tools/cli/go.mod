module github.com/apache/dubbo-go/tools/cli

go 1.13

require (
	github.com/apache/dubbo-go-hessian2 v1.8.0
	github.com/dubbogo/gost v1.11.1
	github.com/pkg/errors v0.9.1
	go.uber.org/atomic v1.7.0
)

replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.4
