module github.com/apache/dubbo-go

go 1.15

require (
	github.com/RoaringBitmap/roaring v0.6.1
	github.com/Workiva/go-datastructures v1.0.52
	github.com/afex/hystrix-go v0.0.0-20180502004556-fa1af6a1f4f5
	github.com/alibaba/sentinel-golang v1.0.2
	github.com/apache/dubbo-getty v1.4.7
	github.com/apache/dubbo-go-hessian2 v1.9.4
	github.com/creasty/defaults v1.5.1
	github.com/dubbogo/go-zookeeper v1.0.3
	github.com/dubbogo/gost v1.11.21-0.20220503144918-9e5ae44480af
	github.com/emicklei/go-restful/v3 v3.4.0
	github.com/fsnotify/fsnotify v1.5.1
	github.com/go-co-op/gocron v0.1.1
	github.com/go-resty/resty/v2 v2.3.0
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/hashicorp/consul v1.8.0
	github.com/hashicorp/consul/api v1.5.0
	github.com/hashicorp/vault/sdk v0.1.14-0.20200519221838-e0cfd64bc267
	github.com/jinzhu/copier v0.0.0-20190625015134-976e0346caa8
	github.com/magiconair/properties v1.8.4
	github.com/mitchellh/mapstructure v1.4.1
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/nacos-group/nacos-sdk-go v1.0.8
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.9.0
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/stretchr/testify v1.7.0
	github.com/zouyx/agollo/v3 v3.4.5
	go.etcd.io/etcd/api/v3 v3.5.0-alpha.0
	go.etcd.io/etcd/client/v3 v3.5.0-alpha.0
	go.uber.org/atomic v1.9.0
	go.uber.org/zap v1.16.0
	google.golang.org/grpc v1.36.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
)

replace (
	github.com/envoyproxy/go-control-plane => github.com/envoyproxy/go-control-plane v0.8.0
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
	k8s.io/api => k8s.io/api v0.16.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.9
	k8s.io/client-go => k8s.io/client-go v0.16.9
)
