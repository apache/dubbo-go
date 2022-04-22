/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mapping

import (
	"encoding/json"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

const (
	debugAdszDataFoo             = `{"totalClients":2,"clients":[{"connectionId":"dubbo-go-app-0.0.1-77b8cd56f9-4hpmb.default-4","connectedAt":"2022-03-23T13:32:49.884373692Z","address":"172.17.80.28:57150","metadata":{"LABELS":{"DUBBO_GO":"{\"providers:api.Greeter::\":\"dubbo-go-app.default.svc.cluster.local:20000\",\"providers:grpc.reflection.v1alpha.ServerReflection::\":\"dubbo-go-app.default.svc.cluster.local:20000\"}","topology.istio.io/cluster":"Kubernetes","topology.kubernetes.io/region":"cn-hangzhou","topology.kubernetes.io/zone":"cn-hangzhou-b"},"CLUSTER_ID":"Kubernetes"},"watches":{"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment":["outbound|443||istiod.istio-system.svc.cluster.local","outbound|443||metrics-server.kube-system.svc.cluster.local","outbound|80||heapster.kube-system.svc.cluster.local","outbound|443||storage-crd-validate-service.kube-system.svc.cluster.local","outbound|20000||laurence-svc.default.svc.cluster.local","outbound|11280||storage-monitor-service.kube-system.svc.cluster.local","outbound|15010||istiod.istio-system.svc.cluster.local","outbound|443||kubernetes.default.svc.cluster.local","outbound|9153||kube-dns.kube-system.svc.cluster.local","outbound|53||kube-dns.kube-system.svc.cluster.local","","outbound|15012||istiod.istio-system.svc.cluster.local","outbound|8848||nacos.default.svc.cluster.local","outbound|15014||istiod.istio-system.svc.cluster.local","outbound|20000||dubbo-go-app.default.svc.cluster.local"]}},{"connectionId":"dubbo-go-app-0.0.2-f465d67f7-k5ckq.default-2","connectedAt":"2022-03-23T13:32:15.790703481Z","address":"172.17.80.29:35406","metadata":{"LABELS":{"DUBBO_GO":"{\"providers:api.Greeter::\":\"dubbo-go-app.default.svc.cluster.local:20000\",\"providers:grpc.reflection.v1alpha.ServerReflection::\":\"dubbo-go-app.default.svc.cluster.local:20000\"}","topology.istio.io/cluster":"Kubernetes","topology.kubernetes.io/region":"cn-hangzhou","topology.kubernetes.io/zone":"cn-hangzhou-b"},"CLUSTER_ID":"Kubernetes"},"watches":{"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment":["outbound|15014||istiod.istio-system.svc.cluster.local","outbound|15010||istiod.istio-system.svc.cluster.local","outbound|53||kube-dns.kube-system.svc.cluster.local","outbound|11280||storage-monitor-service.kube-system.svc.cluster.local","outbound|8848||nacos.default.svc.cluster.local","outbound|20000||dubbo-go-app.default.svc.cluster.local","outbound|9153||kube-dns.kube-system.svc.cluster.local","","outbound|443||storage-crd-validate-service.kube-system.svc.cluster.local","outbound|443||metrics-server.kube-system.svc.cluster.local","outbound|15012||istiod.istio-system.svc.cluster.local","outbound|443||istiod.istio-system.svc.cluster.local","outbound|80||heapster.kube-system.svc.cluster.local","outbound|20000||laurence-svc.default.svc.cluster.local","outbound|443||kubernetes.default.svc.cluster.local"]}}]}`
	debugAdszInvalidDataFoo      = `"totalClients":2,"clients":[{"connectionId":"dubbo-go-app-0.0.1-77b8cd56f9-4hpmb.default-4","connectedAt":"2022-03-23T13:32:49.884373692Z","address":"172.17.80.28:57150","metadata":{"LABELS":{"DUBBO_GO":"{\"providers:api.Greeter::\":\"dubbo-go-app.default.svc.cluster.local:20000\",\"providers:grpc.reflection.v1alpha.ServerReflection::\":\"dubbo-go-app.default.svc.cluster.local:20000\"}","topology.istio.io/cluster":"Kubernetes","topology.kubernetes.io/region":"cn-hangzhou","topology.kubernetes.io/zone":"cn-hangzhou-b"},"CLUSTER_ID":"Kubernetes"},"watches":{"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment":["outbound|443||istiod.istio-system.svc.cluster.local","outbound|443||metrics-server.kube-system.svc.cluster.local","outbound|80||heapster.kube-system.svc.cluster.local","outbound|443||storage-crd-validate-service.kube-system.svc.cluster.local","outbound|20000||laurence-svc.default.svc.cluster.local","outbound|11280||storage-monitor-service.kube-system.svc.cluster.local","outbound|15010||istiod.istio-system.svc.cluster.local","outbound|443||kubernetes.default.svc.cluster.local","outbound|9153||kube-dns.kube-system.svc.cluster.local","outbound|53||kube-dns.kube-system.svc.cluster.local","","outbound|15012||istiod.istio-system.svc.cluster.local","outbound|8848||nacos.default.svc.cluster.local","outbound|15014||istiod.istio-system.svc.cluster.local","outbound|20000||dubbo-go-app.default.svc.cluster.local"]}},{"connectionId":"dubbo-go-app-0.0.2-f465d67f7-k5ckq.default-2","connectedAt":"2022-03-23T13:32:15.790703481Z","address":"172.17.80.29:35406","metadata":{"LABELS":{"DUBBO_GO":"{\"providers:api.Greeter::\":\"dubbo-go-app.default.svc.cluster.local:20000\",\"providers:grpc.reflection.v1alpha.ServerReflection::\":\"dubbo-go-app.default.svc.cluster.local:20000\"}","topology.istio.io/cluster":"Kubernetes","topology.kubernetes.io/region":"cn-hangzhou","topology.kubernetes.io/zone":"cn-hangzhou-b"},"CLUSTER_ID":"Kubernetes"},"watches":{"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment":["outbound|15014||istiod.istio-system.svc.cluster.local","outbound|15010||istiod.istio-system.svc.cluster.local","outbound|53||kube-dns.kube-system.svc.cluster.local","outbound|11280||storage-monitor-service.kube-system.svc.cluster.local","outbound|8848||nacos.default.svc.cluster.local","outbound|20000||dubbo-go-app.default.svc.cluster.local","outbound|9153||kube-dns.kube-system.svc.cluster.local","","outbound|443||storage-crd-validate-service.kube-system.svc.cluster.local","outbound|443||metrics-server.kube-system.svc.cluster.local","outbound|15012||istiod.istio-system.svc.cluster.local","outbound|443||istiod.istio-system.svc.cluster.local","outbound|80||heapster.kube-system.svc.cluster.local","outbound|20000||laurence-svc.default.svc.cluster.local","outbound|443||kubernetes.default.svc.cluster.local"]}}]}`
	debugAdszEmptyDubbogoDataFoo = `{"totalClients":2,"clients":[{"connectionId":"dubbo-go-app-0.0.1-77b8cd56f9-4hpmb.default-4","connectedAt":"2022-03-23T13:32:49.884373692Z","address":"172.17.80.28:57150","metadata":{"LABELS":{"topology.istio.io/cluster":"Kubernetes","topology.kubernetes.io/region":"cn-hangzhou","topology.kubernetes.io/zone":"cn-hangzhou-b"},"CLUSTER_ID":"Kubernetes"},"watches":{"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment":["outbound|443||istiod.istio-system.svc.cluster.local","outbound|443||metrics-server.kube-system.svc.cluster.local","outbound|80||heapster.kube-system.svc.cluster.local","outbound|443||storage-crd-validate-service.kube-system.svc.cluster.local","outbound|20000||laurence-svc.default.svc.cluster.local","outbound|11280||storage-monitor-service.kube-system.svc.cluster.local","outbound|15010||istiod.istio-system.svc.cluster.local","outbound|443||kubernetes.default.svc.cluster.local","outbound|9153||kube-dns.kube-system.svc.cluster.local","outbound|53||kube-dns.kube-system.svc.cluster.local","","outbound|15012||istiod.istio-system.svc.cluster.local","outbound|8848||nacos.default.svc.cluster.local","outbound|15014||istiod.istio-system.svc.cluster.local","outbound|20000||dubbo-go-app.default.svc.cluster.local"]}},{"connectionId":"dubbo-go-app-0.0.2-f465d67f7-k5ckq.default-2","connectedAt":"2022-03-23T13:32:15.790703481Z","address":"172.17.80.29:35406","metadata":{"LABELS":{"topology.istio.io/cluster":"Kubernetes","topology.kubernetes.io/region":"cn-hangzhou","topology.kubernetes.io/zone":"cn-hangzhou-b"},"CLUSTER_ID":"Kubernetes"},"watches":{"type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment":["outbound|15014||istiod.istio-system.svc.cluster.local","outbound|15010||istiod.istio-system.svc.cluster.local","outbound|53||kube-dns.kube-system.svc.cluster.local","outbound|11280||storage-monitor-service.kube-system.svc.cluster.local","outbound|8848||nacos.default.svc.cluster.local","outbound|20000||dubbo-go-app.default.svc.cluster.local","outbound|9153||kube-dns.kube-system.svc.cluster.local","","outbound|443||storage-crd-validate-service.kube-system.svc.cluster.local","outbound|443||metrics-server.kube-system.svc.cluster.local","outbound|15012||istiod.istio-system.svc.cluster.local","outbound|443||istiod.istio-system.svc.cluster.local","outbound|80||heapster.kube-system.svc.cluster.local","outbound|20000||laurence-svc.default.svc.cluster.local","outbound|443||kubernetes.default.svc.cluster.local"]}}]}`
	key1                         = "providers:api.Greeter::"
	val1                         = "dubbo-go-app.default.svc.cluster.local:20000"
	key2                         = "providers:grpc.reflection.v1alpha.ServerReflection::"
	val2                         = "dubbo-go-app.default.svc.cluster.local:20000"
)

func TestADSZResponseGetMap(t *testing.T) {
	adszRsp := &ADSZResponse{}
	assert.Nil(t, json.Unmarshal([]byte(debugAdszDataFoo), adszRsp))

	adszMap := adszRsp.GetMap()
	assert.True(t, len(adszMap) == 2)
	v1, ok1 := adszMap[key1]
	assert.True(t, ok1)
	assert.Equal(t, val1, v1)

	v2, ok2 := adszMap[key2]
	assert.True(t, ok2)
	assert.Equal(t, val2, v2)
}

func TestInvalidADSZResponseGetMap(t *testing.T) {
	adszRsp := &ADSZResponse{}
	json.Unmarshal([]byte(debugAdszInvalidDataFoo), adszRsp)

	adszMap := adszRsp.GetMap()
	assert.True(t, len(adszMap) == 0)
}

func TestEmptyDubbogoMapADSZResponseGetMap(t *testing.T) {
	adszRsp := &ADSZResponse{}
	assert.Nil(t, json.Unmarshal([]byte(debugAdszEmptyDubbogoDataFoo), adszRsp))

	adszMap := adszRsp.GetMap()
	assert.True(t, len(adszMap) == 0)
}
