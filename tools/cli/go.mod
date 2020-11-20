module github.com/apache/dubbo-go/tools/cli

go 1.13

require (
	github.com/LaurenceLiZhixin/dubbo-go-hessian2 v1.7.3
	github.com/LaurenceLiZhixin/json-interface-parser v0.0.0-20201026115035-e5b01058601f
	github.com/pkg/errors v0.9.1
	go.uber.org/atomic v1.7.0
)

replace github.com/LaurenceLiZhixin/dubbo-go-hessian2 v1.7.3 => ../../../dubbo-go-hessian2

replace github.com/LaurenceLiZhixin/json-interface-parser v0.0.0-20201026115035-e5b01058601f => ../../../json-interface-parser
