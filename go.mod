module github.com/apache/dubbo-go

go 1.14

require (
	github.com/apache/dubbo-go-hessian2 v1.2.5-0.20190731020727-1697039810c8
	github.com/dubbogo/getty v1.2.2
	github.com/dubbogo/gost v1.9.0
	github.com/magiconair/properties v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/samuel/go-zookeeper v0.0.0-20180130194729-c4fab1ac1bec
	github.com/stretchr/testify v1.5.1
	go.uber.org/atomic v1.6.0
	go.uber.org/zap v1.15.0
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/dubbogo/getty v1.2.2 => github.com/aliiohs/getty v1.1.1-0.20200802094147-169328c4ff38
