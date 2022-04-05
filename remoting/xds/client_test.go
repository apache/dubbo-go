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

package xds

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
	registryMocks "dubbo.apache.org/dubbo-go/v3/registry/mocks"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds/common"
	"dubbo.apache.org/dubbo-go/v3/xds/client"
	"dubbo.apache.org/dubbo-go/v3/xds/client/mocks"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

const (
	dubbogoPortFoo  = "20000"
	istioXDSPortFoo = "15010"

	podNameFoo        = "mockPodName"
	localNamespaceFoo = "default"

	localIPFoo = "172.16.100.1"
	istioIPFoo = "172.16.100.2"

	localAddrFoo    = localIPFoo + ":" + dubbogoPortFoo
	istioXDSAddrFoo = istioIPFoo + ":" + istioXDSPortFoo

	istioHostNameFoo = "istiod.istio-system.svc.cluster.local"
	istioHostAddrFoo = istioHostNameFoo + ":" + istioXDSPortFoo
	localHostNameFoo = "dubbo-go-app." + localNamespaceFoo + ".svc.cluster.local"
	localHostAddrFoo = localHostNameFoo + ":" + dubbogoPortFoo

	istioClusterNameFoo = "outbound|" + istioXDSPortFoo + "||" + istioHostNameFoo
	localClusterNameFoo = "outbound|" + dubbogoPortFoo + "||" + localHostNameFoo
)

const (
	providerIPFoo     = "172.16.100.3"
	providerV1IPFoo   = "172.16.100.4"
	providerV2IPFoo   = "172.16.100.5"
	providerAddrFoo   = providerIPFoo + ":" + dubbogoPortFoo
	providerV1AddrFoo = providerV1IPFoo + ":" + dubbogoPortFoo
	providerV2AddrFoo = providerV2IPFoo + ":" + dubbogoPortFoo

	dubbogoProviderHostName    = "dubbo-go-app-provider." + localNamespaceFoo + ".svc.cluster.local"
	dubbogoProviderHostAddrFoo = dubbogoProviderHostName + ":" + dubbogoPortFoo

	dubbogoProviderInterfaceNameFoo    = "api.Greeter"
	dubbogoProviderserivceUniqueKeyFoo = "provider::" + dubbogoProviderInterfaceNameFoo

	providerClusterNameFoo   = "outbound|" + dubbogoPortFoo + "||" + dubbogoProviderHostName
	providerClusterV1NameFoo = "outbound|" + dubbogoPortFoo + "|v1|" + dubbogoProviderHostName
	providerClusterV2NameFoo = "outbound|" + dubbogoPortFoo + "|v2|" + dubbogoProviderHostName
)

func TestWrappedClientImpl(t *testing.T) {
	// test New WrappedClientImpl
	testFailedWithIstioCDS(t)
	testFailedWithLocalCDS(t)
	testFailedWithNoneCDS(t)

	testFailedWithLocalEDSFailed(t)
	testFailedWithIstioEDSFailed(t)

	testWithDiscoverySuccess(t)

	assert.NotNil(t, GetXDSWrappedClient())

	// test Subscription
	testSubscribe(t)
}

func testWithDiscoverySuccess(t *testing.T) {
	mockXDSClient := &mocks.XDSClient{}
	cancelCalledCounter := &atomic.Int32{}
	// all cluster CDS
	mockXDSClient.On("WatchCluster", "*",
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.ClusterUpdate, err error)) bool {
			// istioClusterName CDS, this and next calling must be async because testify.Mock.MethodCalled() has calling lock
			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: istioClusterNameFoo,
			}, nil)

			// localClusterName CDS
			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: localClusterNameFoo,
			}, nil)
			return true
		})).
		Return(func() {})

	// istio cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			return clusterName == istioClusterNameFoo
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: istioXDSAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {
			// return cancel function
			cancelCalledCounter.Inc()
		})

	// local cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			return clusterName == localClusterNameFoo
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: localAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {
			// return cancel function
			cancelCalledCounter.Inc()
		})

	xdsClientFactoryFunction = func(localIP, podName, namespace string, istioAddr common.HostAddr) (client.XDSClient, error) {
		return mockXDSClient, nil
	}
	xdsWrappedClient, err := NewXDSWrappedClient(podNameFoo, localNamespaceFoo, localIPFoo, common.NewHostNameOrIPAddr(istioHostAddrFoo))
	assert.Nil(t, err)
	assert.NotNil(t, xdsWrappedClient)

	// assert eds cancel is called
	assert.Equal(t, int32(2), cancelCalledCounter.Load())
	// discovery p
	assert.Equal(t, istioIPFoo, xdsWrappedClient.GetIstioPodIP())
	address := xdsWrappedClient.GetHostAddress()
	assert.Equal(t, localHostAddrFoo, address.String())
}

func testFailedWithIstioCDS(t *testing.T) {
	mockXDSClient := &mocks.XDSClient{}
	cancelCalledCounter := &atomic.Int32{}
	// all cluster CDS
	mockXDSClient.On("WatchCluster", "*",
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.ClusterUpdate, err error)) bool {
			// do not send istioClusterName message, which cause discover istio ip failed
			//go clusterUpdateHandler(resource.ClusterUpdate{
			//	ClusterName: istioClusterNameFoo,
			//}, nil)

			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: localClusterNameFoo,
			}, nil)
			return true
		})).
		Return(func() {})

	// istio cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			return clusterName == istioClusterNameFoo
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: istioXDSAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {
			// return cancel function
			cancelCalledCounter.Inc()
		})

	// local cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			return clusterName == localClusterNameFoo
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: localAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {
			// return cancel function
			cancelCalledCounter.Inc()
		})

	xdsClientFactoryFunction = func(localIP, podName, namespace string, istioAddr common.HostAddr) (client.XDSClient, error) {
		return mockXDSClient, nil
	}
	xdsWrappedClient, err := NewXDSWrappedClient(podNameFoo, localNamespaceFoo, localIPFoo, common.NewHostNameOrIPAddr(istioHostAddrFoo))
	assert.Equal(t, DiscoverIstioPodError, err)
	assert.Nil(t, xdsWrappedClient)
	assert.Equal(t, int32(1), cancelCalledCounter.Load())
}

func testFailedWithLocalCDS(t *testing.T) {
	mockXDSClient := &mocks.XDSClient{}
	cancelCalledCounter := &atomic.Int32{}
	// all cluster CDS
	mockXDSClient.On("WatchCluster", "*",
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.ClusterUpdate, err error)) bool {
			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: istioClusterNameFoo,
			}, nil)

			// do not send localClusterNameFoo cds message, which cause discover local addr failed
			//go clusterUpdateHandler(resource.ClusterUpdate{
			//	ClusterName: localClusterNameFoo,
			//}, nil)
			return true
		})).
		Return(func() {})

	// istio cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			return clusterName == istioClusterNameFoo
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: istioXDSAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {
			// return cancel function
			cancelCalledCounter.Inc()
		})

	// local cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			return clusterName == localClusterNameFoo
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: localAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {
			// return cancel function
			cancelCalledCounter.Inc()
		})

	xdsClientFactoryFunction = func(localIP, podName, namespace string, istioAddr common.HostAddr) (client.XDSClient, error) {
		return mockXDSClient, nil
	}
	xdsWrappedClient, err := NewXDSWrappedClient(podNameFoo, localNamespaceFoo, localIPFoo, common.NewHostNameOrIPAddr(istioHostAddrFoo))
	assert.Equal(t, DiscoverLocalError, err)
	assert.Nil(t, xdsWrappedClient)
	assert.Equal(t, int32(1), cancelCalledCounter.Load())
}

func testFailedWithNoneCDS(t *testing.T) {
	mockXDSClient := &mocks.XDSClient{}
	cancelCalledCounter := &atomic.Int32{}
	// all cluster CDS
	mockXDSClient.On("WatchCluster", "*",
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.ClusterUpdate, err error)) bool {
			// do not send any cds message, which cause discover failed
			//go clusterUpdateHandler(resource.ClusterUpdate{
			//	ClusterName: istioClusterNameFoo,
			//}, nil)

			//go clusterUpdateHandler(resource.ClusterUpdate{
			//	ClusterName: localClusterNameFoo,
			//}, nil)
			return true
		})).
		Return(func() {})

	// istio cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			return clusterName == istioClusterNameFoo
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: istioXDSAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {
			// return cancel function
			cancelCalledCounter.Inc()
		})

	// local cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			return clusterName == localClusterNameFoo
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: localAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {
			// return cancel function
			cancelCalledCounter.Inc()
		})

	xdsClientFactoryFunction = func(localIP, podName, namespace string, istioAddr common.HostAddr) (client.XDSClient, error) {
		return mockXDSClient, nil
	}
	xdsWrappedClient, err := NewXDSWrappedClient(podNameFoo, localNamespaceFoo, localIPFoo, common.NewHostNameOrIPAddr(istioHostAddrFoo))
	assert.Equal(t, DiscoverIstioPodError, err)
	assert.Nil(t, xdsWrappedClient)
	assert.Equal(t, int32(0), cancelCalledCounter.Load())
}

func testFailedWithLocalEDSFailed(t *testing.T) {
	mockXDSClient := &mocks.XDSClient{}
	cancelCalledCounter := &atomic.Int32{}
	// all cluster CDS
	mockXDSClient.On("WatchCluster", "*",
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.ClusterUpdate, err error)) bool {
			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: istioClusterNameFoo,
			}, nil)

			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: localClusterNameFoo,
			}, nil)
			return true
		})).
		Return(func() {})

	// istio cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			return clusterName == istioClusterNameFoo
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: istioXDSAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {
			// return cancel function
			cancelCalledCounter.Inc()
		})

	// local cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			return clusterName == localClusterNameFoo
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			// do not send local eds message
			//clusterUpdateHandler(resource.EndpointsUpdate{
			//	Localities: []resource.Locality{
			//		{
			//			Endpoints: []resource.Endpoint{
			//				{
			//					Address: localAddrFoo,
			//				},
			//			},
			//		},
			//	},
			//}, nil)
			return true
		})).
		Return(func() {
			// return cancel function
			cancelCalledCounter.Inc()
		})

	xdsClientFactoryFunction = func(localIP, podName, namespace string, istioAddr common.HostAddr) (client.XDSClient, error) {
		return mockXDSClient, nil
	}
	xdsWrappedClient, err := NewXDSWrappedClient(podNameFoo, localNamespaceFoo, localIPFoo, common.NewHostNameOrIPAddr(istioHostAddrFoo))
	assert.Equal(t, DiscoverLocalError, err)
	assert.Nil(t, xdsWrappedClient)
	assert.Equal(t, int32(2), cancelCalledCounter.Load())
}

func testFailedWithIstioEDSFailed(t *testing.T) {
	mockXDSClient := &mocks.XDSClient{}
	cancelCalledCounter := &atomic.Int32{}
	// all cluster CDS
	mockXDSClient.On("WatchCluster", "*",
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.ClusterUpdate, err error)) bool {
			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: istioClusterNameFoo,
			}, nil)

			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: localClusterNameFoo,
			}, nil)
			return true
		})).
		Return(func() {})

	// istio cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			return clusterName == istioClusterNameFoo
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			// do not send istio eds message
			//clusterUpdateHandler(resource.EndpointsUpdate{
			//	Localities: []resource.Locality{
			//		{
			//			Endpoints: []resource.Endpoint{
			//				{
			//					Address: istioXDSAddrFoo,
			//				},
			//			},
			//		},
			//	},
			//}, nil)
			return true
		})).
		Return(func() {
			// return cancel function
			cancelCalledCounter.Inc()
		})

	// local cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			return clusterName == localClusterNameFoo
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: localAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {
			// return cancel function
			cancelCalledCounter.Inc()
		})

	xdsClientFactoryFunction = func(localIP, podName, namespace string, istioAddr common.HostAddr) (client.XDSClient, error) {
		return mockXDSClient, nil
	}
	xdsWrappedClient, err := NewXDSWrappedClient(podNameFoo, localNamespaceFoo, localIPFoo, common.NewHostNameOrIPAddr(istioHostAddrFoo))
	assert.Equal(t, DiscoverIstioPodError, err)
	assert.Nil(t, xdsWrappedClient)
	assert.Equal(t, int32(2), cancelCalledCounter.Load())
}

func testSubscribe(t *testing.T) {
	mockXDSClient := &mocks.XDSClient{}
	// all cluster CDS
	mockXDSClient.On("WatchCluster", "*",
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.ClusterUpdate, err error)) bool {
			// istio ClusterName CDS, this and next calling must be async because testify.Mock.MethodCalled() has calling lock
			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: istioClusterNameFoo,
			}, nil)

			// local ClusterName CDS
			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: localClusterNameFoo,
			}, nil)

			// provider Cluster Name CDS
			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: providerClusterNameFoo,
			}, nil)

			// provider Cluster Name v1 CDS
			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: providerClusterV1NameFoo,
			}, nil)

			// provider Cluster Name v2 CDS
			go clusterUpdateHandler(resource.ClusterUpdate{
				ClusterName: providerClusterV2NameFoo,
			}, nil)
			return true
		})).
		Return(func() {})

	// istio cluster EDS
	clusterMatch := false
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			clusterMatch = clusterName == istioClusterNameFoo
			return clusterMatch
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			if !clusterMatch {
				return false
			}
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: istioXDSAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {})

	// local cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			clusterMatch = clusterName == localClusterNameFoo
			return clusterMatch
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			if !clusterMatch {
				return false
			}
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: localAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {})

	// provider cluster EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			clusterMatch = clusterName == providerClusterNameFoo
			return clusterMatch
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			if !clusterMatch {
				return false
			}
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: providerAddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {})

	// provider cluster v1 EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			clusterMatch = clusterName == providerClusterV1NameFoo
			return clusterMatch
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			if !clusterMatch {
				return false
			}
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: providerV1AddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {})

	// todo test router RDS
	mockXDSClient.On("WatchRouteConfig", mock.Anything, mock.Anything).Return(func() {})

	// provider cluster v2 EDS
	mockXDSClient.On("WatchEndpoints",
		mock.MatchedBy(func(clusterName string) bool {
			clusterMatch = clusterName == providerClusterV2NameFoo
			return clusterMatch
		}),
		mock.MatchedBy(func(clusterUpdateHandler func(update resource.EndpointsUpdate, err error)) bool {
			if !clusterMatch {
				return false
			}
			clusterUpdateHandler(resource.EndpointsUpdate{
				Localities: []resource.Locality{
					{
						Endpoints: []resource.Endpoint{
							{
								Address: providerV2AddrFoo,
							},
						},
					},
				},
			}, nil)
			return true
		})).
		Return(func() {})

	xdsClientFactoryFunction = func(localIP, podName, namespace string, istioAddr common.HostAddr) (client.XDSClient, error) {
		return mockXDSClient, nil
	}

	xdsWrappedClient = nil
	xdsWrappedClient, err := NewXDSWrappedClient(podNameFoo, localNamespaceFoo, localIPFoo, common.NewHostNameOrIPAddr(istioHostAddrFoo))
	assert.Nil(t, err)
	assert.NotNil(t, xdsWrappedClient)

	notifyListener := &registryMocks.NotifyListener{}
	notifyListener.On("Notify", mock.Anything).Return(nil)

	go xdsWrappedClient.Subscribe(dubbogoProviderserivceUniqueKeyFoo, dubbogoProviderInterfaceNameFoo, dubbogoProviderHostAddrFoo, notifyListener)

	time.Sleep(time.Second)
	notifyListener.AssertCalled(t, "Notify", mock.MatchedBy(func(event *registry.ServiceEvent) bool {
		if event.Action == remoting.EventTypeUpdate &&
			event.Service.Ip == providerV2IPFoo &&
			event.Service.GetParam(constant.MeshClusterIDKey, "") == providerClusterV2NameFoo {
			return true
		}
		return false
	}))

	notifyListener.AssertCalled(t, "Notify", mock.MatchedBy(func(event *registry.ServiceEvent) bool {
		if event.Action == remoting.EventTypeUpdate &&
			event.Service.Ip == providerV1IPFoo &&
			event.Service.GetParam(constant.MeshClusterIDKey, "") == providerClusterV1NameFoo {
			return true
		}
		return false
	}))

	notifyListener.AssertCalled(t, "Notify", mock.MatchedBy(func(event *registry.ServiceEvent) bool {
		if event.Action == remoting.EventTypeUpdate &&
			event.Service.Ip == providerIPFoo &&
			event.Service.GetParam(constant.MeshClusterIDKey, "") == providerClusterNameFoo {
			return true
		}
		return false
	}))
}

// todo TestDestroy
// todo TestRDS
