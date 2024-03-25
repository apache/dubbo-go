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

package channel

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"github.com/dubbogo/gost/log/logger"
	v3configcore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	v3resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type XdsUpdateListener func(*v3discovery.DiscoveryResponse, *XdsClientChannel) error

type ResourceType int

const (
	ListenerType ResourceType = iota
	RouteType
	ClusterType
	EndpointType
	UnSupportType
)

type XdsClientChannel struct {
	udsPath         string
	conn            *grpc.ClientConn
	cancel          context.CancelFunc
	stopChan        chan struct{}
	updateChan      chan *v3discovery.DiscoveryResponse
	adsClient       v3discovery.AggregatedDiscoveryServiceClient
	streamAdsClient v3discovery.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	node            *v3configcore.Node
	listeners       map[ResourceType]map[string]XdsUpdateListener
	listenerMutex   sync.RWMutex
	ApiStore        *ApiStore
	// stop or not
	runningStatus atomic.Bool
}

func NewXdsClientChannel(stopChan chan struct{}, xdsUdsPath string, node *v3configcore.Node) (*XdsClientChannel, error) {
	udsPath := "unix:" + xdsUdsPath
	conn, err := grpc.Dial(
		udsPath,
		grpc.WithInsecure(),
	)
	if err != nil {
		logger.Errorf("xds.subscribe.stream", "[xds][subscribe] dial grpc server failed: %v", err)
		return nil, err
	}

	adsClient := v3discovery.NewAggregatedDiscoveryServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())

	xdsClient := &XdsClientChannel{
		udsPath:         udsPath,
		conn:            conn,
		cancel:          cancel,
		node:            node,
		adsClient:       adsClient,
		listeners:       make(map[ResourceType]map[string]XdsUpdateListener),
		listenerMutex:   sync.RWMutex{},
		stopChan:        stopChan,
		updateChan:      make(chan *v3discovery.DiscoveryResponse, 4),
		streamAdsClient: nil,
		ApiStore:        NewApiStore(),
	}

	if xdsClient.streamAdsClient, err = adsClient.StreamAggregatedResources(ctx); err != nil {
		logger.Errorf("xds.subscribe.stream", "[xds][subscribe] get ADS stream fail: %v", err)
		conn.Close()
		return nil, err
	}

	go xdsClient.startListeningAndProcessingUpdates()

	return xdsClient, nil
}

func (xds *XdsClientChannel) Send(req *v3discovery.DiscoveryRequest) error {
	if req == nil {
		return nil
	}
	logger.Infof("xds send xds request = %s ", utils.GetJsonString(req))
	return xds.streamAdsClient.Send(req)
}

func (xds *XdsClientChannel) SendWithTypeUrlAndResourceNames(typeUrl string, resourceNames []string) error {
	request := &v3discovery.DiscoveryRequest{
		VersionInfo:   "",
		ResourceNames: resourceNames,
		TypeUrl:       typeUrl,
		ResponseNonce: "",
		ErrorDetail:   nil,
		Node:          xds.node,
	}
	logger.Infof("xds send xds request typeurl = %s request = %s ", typeUrl, utils.GetJsonString(request))
	if err := xds.streamAdsClient.Send(request); err != nil {
		logger.Errorf("send typeurl %s with resourceNames %v failed, error: %v", typeUrl, resourceNames, err)
		return err
	}
	return nil
}

func (xds *XdsClientChannel) AckResponse(resp *v3discovery.DiscoveryResponse) {
	info := xds.ApiStore.Find(resp.TypeUrl)
	ack := &v3discovery.DiscoveryRequest{
		VersionInfo:   resp.VersionInfo,
		ResourceNames: info.ResourceNames,
		TypeUrl:       resp.TypeUrl,
		ResponseNonce: resp.Nonce,
		ErrorDetail:   nil,
		Node:          xds.node,
	}
	logger.Infof("xds send ack response = %s ", utils.GetJsonString(ack))
	if err := xds.streamAdsClient.Send(ack); err != nil {
		logger.Errorf("response %s ack failed, error: %v", resp.TypeUrl, err)
	}
}

func (xds *XdsClientChannel) startListeningAndProcessingUpdates() {
	xds.runningStatus.Store(true)
	go xds.listenForResourceUpdates()
	go func() {
		for {
			select {
			case <-xds.stopChan:
				xds.Stop()
				return
			default:
			}

			if xds.streamAdsClient == nil {
				continue
			}
			resp, err := xds.streamAdsClient.Recv()
			if err != nil {
				st, ok := status.FromError(err)
				if ok && st.Code() == codes.Canceled {
					logger.Infof("[xds channel] xds channel context was canceled")
					return
				}
				//if err != nil && err != io.EOF {
				logger.Errorf("xds.recv.error", "[xds][recv] error receiving resources: %v", err)
				if err2 := xds.reconnect(); err2 != nil {
					logger.Errorf("xds.reconnect.error", "[xds][reconnect] failed to reconnect: %v", err2)
					continue
				} else {
					// TODO need to subscribe all resources again!!!
				}
				continue
			}

			//if err == io.EOF {
			//	continue
			//}

			logger.Infof("xds recv resp = %s", utils.ConvertResponseToString(resp))
			if resp.GetTypeUrl() == v3resource.ListenerType || resp.GetTypeUrl() == v3resource.RouteType ||
				resp.GetTypeUrl() == v3resource.ClusterType || resp.GetTypeUrl() == v3resource.EndpointType {
				xds.updateChan <- resp
			}
			// TODO need to ack response
		}
	}()

	<-xds.stopChan
}

func (xds *XdsClientChannel) reconnect() error {
	xds.closeConnection()

	select {
	case <-time.After(1 * time.Second):
		logger.Infof("delay 1 seconds to reconnect sds server")
		break
	}

	newConn, err := grpc.Dial(
		xds.udsPath,
		grpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("[xds][reconnect] dial grpc server failed: %w", err)
	}

	xds.conn = newConn
	xds.adsClient = v3discovery.NewAggregatedDiscoveryServiceClient(newConn)
	ctx, cancel := context.WithCancel(context.Background())
	xds.cancel = cancel

	if xds.streamAdsClient, err = xds.adsClient.StreamAggregatedResources(ctx); err != nil {
		return fmt.Errorf("[xds][reconnect] get ADS stream fail: %w", err)
	}

	return nil
}

func (xds *XdsClientChannel) listenForResourceUpdates() {
	for {
		select {
		case <-xds.stopChan:
			return
		case resp, ok := <-xds.updateChan:
			if !ok {
				continue
			}

			resourceType := getResourceTypeFromTypeUrl(resp.GetTypeUrl())
			if resourceType == UnSupportType {
				continue
			}

			func() {
				xds.listenerMutex.RLock()
				defer xds.listenerMutex.RUnlock()
				for key, listener := range xds.listeners[resourceType] {
					if err := listener(resp, xds); err != nil {
						logger.Errorf("xds.listener.error [xds][listener:%s] failed to process resource update: %v", key, err)
					}

				}
			}()
		}
	}
}

func getResourceTypeFromTypeUrl(typeUrl string) ResourceType {
	switch typeUrl {
	case v3resource.ListenerType:
		return ListenerType
	case v3resource.RouteType:
		return RouteType
	case v3resource.ClusterType:
		return ClusterType
	case v3resource.EndpointType:
		return EndpointType
	default:
		logger.Errorf("Unsupported resource type: %d", typeUrl)
		return UnSupportType
	}
}

func (xds *XdsClientChannel) AddListener(listener XdsUpdateListener, key string, resourceType ResourceType) {
	xds.listenerMutex.Lock()
	defer xds.listenerMutex.Unlock()

	if _, ok := xds.listeners[resourceType]; !ok {
		xds.listeners[resourceType] = make(map[string]XdsUpdateListener)
	}
	xds.listeners[resourceType][key] = listener
}

func (xds *XdsClientChannel) RemoveListener(key string, resourceType ResourceType) {
	xds.listenerMutex.Lock()
	defer xds.listenerMutex.Unlock()
	if listenerMap, ok := xds.listeners[resourceType]; ok {
		delete(listenerMap, key)
	}
}

func (xds *XdsClientChannel) closeConnection() {
	xds.cancel()
	if xds.conn != nil {
		xds.conn.Close()
		xds.conn = nil
	}
}

func (xds *XdsClientChannel) Stop() {
	if runningStatus := xds.runningStatus.Load(); runningStatus {
		// make sure stop once
		xds.runningStatus.Store(false)
		logger.Infof("[xds channel] Stop now...")
		xds.closeConnection()
		close(xds.updateChan)
	}

}

func (xds *XdsClientChannel) InitXds() error {
	xds.Send(xds.InitClusterRequest())
	xds.Send(xds.CreateLdsRequest())
	return nil
}

func (xds *XdsClientChannel) InitClusterRequest() *v3discovery.DiscoveryRequest {
	return &v3discovery.DiscoveryRequest{
		VersionInfo:   "",
		ResourceNames: []string{},
		TypeUrl:       EnvoyCluster,
		ResponseNonce: "",
		ErrorDetail:   nil,
		Node:          xds.node,
	}
}

func (xds *XdsClientChannel) CreateLdsRequest() *v3discovery.DiscoveryRequest {
	info := xds.ApiStore.Find(EnvoyListener)
	return &v3discovery.DiscoveryRequest{
		VersionInfo:   info.VersionInfo,
		ResourceNames: info.ResourceNames,
		TypeUrl:       EnvoyListener,
		ResponseNonce: info.ResponseNonce,
		ErrorDetail:   nil,
		Node:          xds.node,
	}
}

func (xds *XdsClientChannel) CreateCdsRequest() *v3discovery.DiscoveryRequest {
	info := xds.ApiStore.Find(EnvoyCluster)
	return &v3discovery.DiscoveryRequest{
		VersionInfo:   info.VersionInfo,
		ResourceNames: info.ResourceNames,
		TypeUrl:       EnvoyCluster,
		ResponseNonce: info.ResponseNonce,
		ErrorDetail:   nil,
		Node:          xds.node,
	}
}

func (xds *XdsClientChannel) CreateRdsRequest() *v3discovery.DiscoveryRequest {
	info := xds.ApiStore.Find(EnvoyRoute)
	return &v3discovery.DiscoveryRequest{
		VersionInfo:   info.VersionInfo,
		ResourceNames: info.ResourceNames,
		TypeUrl:       EnvoyRoute,
		ResponseNonce: info.ResponseNonce,
		ErrorDetail:   nil,
		Node:          xds.node,
	}
}

func (xds *XdsClientChannel) CreateEdsRequest() *v3discovery.DiscoveryRequest {
	info := xds.ApiStore.Find(EnvoyEndpoint)
	return &v3discovery.DiscoveryRequest{
		VersionInfo:   info.VersionInfo,
		ResourceNames: info.ResourceNames,
		TypeUrl:       EnvoyEndpoint,
		ResponseNonce: info.ResponseNonce,
		ErrorDetail:   nil,
		Node:          xds.node,
	}
}
