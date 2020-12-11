module github.com/apache/dubbo-go

require (
	cloud.google.com/go v0.50.0 // indirect
	github.com/Microsoft/go-winio v0.4.15-0.20190919025122-fc70bd9a86b5 // indirect
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/RoaringBitmap/roaring v0.5.5
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/Workiva/go-datastructures v1.0.52
	github.com/afex/hystrix-go v0.0.0-20180502004556-fa1af6a1f4f5
	github.com/alibaba/sentinel-golang v0.6.1
	github.com/apache/dubbo-getty v1.3.10
	github.com/apache/dubbo-go-hessian2 v1.7.0
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/etcd v3.3.25+incompatible
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/creasty/defaults v1.5.1
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/dubbogo/go-zookeeper v1.0.2
	github.com/dubbogo/gost v1.10.0
	github.com/elazarl/go-bindata-assetfs v1.0.0 // indirect
	github.com/emicklei/go-restful/v3 v3.0.0
	github.com/frankban/quicktest v1.4.1 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-co-op/gocron v0.1.1
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-resty/resty/v2 v2.3.0
	github.com/gogo/protobuf v1.3.1
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.0
	github.com/google/go-cmp v0.4.0 // indirect
	github.com/gophercloud/gophercloud v0.3.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.5 // indirect
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/hashicorp/consul v1.8.0
	github.com/hashicorp/consul/api v1.5.0
	github.com/hashicorp/go-raftchunking v0.6.3-0.20191002164813-7e9e8525653a // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/hashicorp/vault/api v1.0.5-0.20191108163347-bdd38fca2cff // indirect
	github.com/hashicorp/vault/sdk v0.1.14-0.20191112033314-390e96e22eb2
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/jinzhu/copier v0.0.0-20190625015134-976e0346caa8
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/linode/linodego v0.10.0 // indirect
	github.com/magiconair/properties v1.8.1
	github.com/miekg/dns v1.1.27 // indirect
	github.com/mitchellh/hashstructure v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.3.3
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/nacos-group/nacos-sdk-go v1.0.1
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pierrec/lz4 v2.2.6+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/tmc/grpc-websocket-proxy v0.0.0-20200122045848-3419fae592fc // indirect
	github.com/zouyx/agollo/v3 v3.4.5
	go.etcd.io/bbolt v1.3.4 // indirect
	go.uber.org/atomic v1.6.0
	go.uber.org/zap v1.16.0
	golang.org/x/net v0.0.0-20200822124328-c89045814202
	golang.org/x/sys v0.0.0-20200523222454-059865788121 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/grpc v1.26.0
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.16.9
	k8s.io/apimachinery v0.16.9
	k8s.io/client-go v0.16.9
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a // indirect
)

replace (
	github.com/envoyproxy/go-control-plane => github.com/envoyproxy/go-control-plane v0.8.0
	github.com/shirou/gopsutil => github.com/shirou/gopsutil v0.0.0-20181107111621-48177ef5f880
	launchpad.net/gocheck => github.com/go-check/check v0.0.0-20140225173054-eb6ee6f84d0a
)

go 1.13
