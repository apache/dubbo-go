module github.com/apache/dubbo-go

go 1.13

require (
	github.com/RoaringBitmap/roaring v0.5.5
	github.com/Workiva/go-datastructures v1.0.52
	github.com/afex/hystrix-go v0.0.0-20180502004556-fa1af6a1f4f5
	github.com/alibaba/sentinel-golang v1.0.2
	github.com/apache/dubbo-getty v1.4.3
	github.com/apache/dubbo-go-hessian2 v1.8.2
	github.com/coreos/etcd v3.3.25+incompatible
	github.com/creasty/defaults v1.5.1
	github.com/dubbogo/go-zookeeper v1.0.3
	github.com/dubbogo/gost v1.11.2
	github.com/emicklei/go-restful/v3 v3.4.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-co-op/gocron v0.1.1
	github.com/go-resty/resty/v2 v2.3.0
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/hashicorp/consul v1.8.0
	github.com/hashicorp/consul/api v1.5.0
	github.com/hashicorp/vault/sdk v0.1.14-0.20191112033314-390e96e22eb2
	github.com/jinzhu/copier v0.0.0-20190625015134-976e0346caa8
	github.com/magiconair/properties v1.8.4
	github.com/mitchellh/mapstructure v1.4.1
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/nacos-group/nacos-sdk-go v1.0.6
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.9.0
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/stretchr/testify v1.7.0
	github.com/zouyx/agollo/v3 v3.4.5
	go.uber.org/atomic v1.7.0
	go.uber.org/zap v1.16.0
	golang.org/x/sys v0.0.0-20201223074533-0d417f636930 // indirect
	google.golang.org/grpc v1.33.1
	google.golang.org/grpc/examples v0.0.0-20210301210255-fc8f38cccf75 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.16.9
	k8s.io/apimachinery v0.16.9
	k8s.io/client-go v0.16.9
)

replace (
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.4
	github.com/envoyproxy/go-control-plane => github.com/envoyproxy/go-control-plane v0.8.0
	github.com/shirou/gopsutil => github.com/shirou/gopsutil v0.0.0-20181107111621-48177ef5f880
	go.etcd.io/bbolt v1.3.4 => github.com/coreos/bbolt v1.3.3
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
