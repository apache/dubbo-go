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

	"github.com/dubbogo/gost/log/logger"
	v3configcore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	v3discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	v3secret "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	DefaultSecretResourceNames = []string{"default", "ROOTCA"}
)

type SecretUpdateListener func(secret *tls.Secret) error

type SdsClientChannel struct {
	udsPath               string
	conn                  *grpc.ClientConn
	cancel                context.CancelFunc
	stopChan              chan struct{}
	updateChan            chan *tls.Secret
	secretDiscoveryClient v3secret.SecretDiscoveryServiceClient
	streamSecretsClient   v3secret.SecretDiscoveryService_StreamSecretsClient
	node                  *v3configcore.Node
	listeners             map[string]SecretUpdateListener
	listenerMutex         sync.RWMutex
	// stop or not
	runningStatus atomic.Bool
}

func NewSdsClientChannel(stopChan chan struct{}, sdsUdsPath string, node *v3configcore.Node) (*SdsClientChannel, error) {
	udsPath := "unix:" + sdsUdsPath
	conn, err := grpc.Dial(
		udsPath,
		grpc.WithInsecure(),
	)
	if err != nil {
		logger.Errorf("[sds channel] dial grpc server failed %v", err)
		return nil, err
	}
	sdsServiceClient := v3secret.NewSecretDiscoveryServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	sdsStreamClient := &SdsClientChannel{
		udsPath:               udsPath,
		conn:                  conn,
		cancel:                cancel,
		node:                  node,
		secretDiscoveryClient: sdsServiceClient,
		updateChan:            make(chan *tls.Secret, 4),
		stopChan:              stopChan,
		listeners:             make(map[string]SecretUpdateListener),
		listenerMutex:         sync.RWMutex{},
	}
	sdsStreamClient.streamSecretsClient, err = sdsServiceClient.StreamSecrets(ctx)
	if err != nil {
		logger.Errorf("[sds channel] sds.subscribe.stream get sds stream secret fail %v", err)
		conn.Close()
		return nil, err
	}
	go sdsStreamClient.startListening()
	return sdsStreamClient, nil
}

func (sds *SdsClientChannel) startListening() {
	sds.runningStatus.Store(true)
	go func() {
		// need to subscribe all resources
		//if err := sds.Send([]string{"default", "ROOTCA"}); err != nil {
		//	logger.Errorf("sds.send.error [sds][send] failed: %v", err)
		//}
		for {
			select {
			case <-sds.stopChan:
				sds.Stop()
				return
			default:

			}

			if sds.streamSecretsClient == nil {
				continue
			}
			resp, err := sds.streamSecretsClient.Recv()
			if err != nil {
				st, ok := status.FromError(err)
				if ok && st.Code() == codes.Canceled {
					logger.Infof("[sds channel] sds channel context was canceled")
					return
				}
				logger.Errorf("[sds channel] sds.recv.error error: %v", err)
				if err := sds.reconnect(); err != nil {
					logger.Errorf("[sds channel] sds.reconnect.error: %v", err)
					continue
				} else {
					// need to subscribe all resources again
					if err2 := sds.Send(DefaultSecretResourceNames); err2 != nil {
						logger.Errorf("[sds channel] sds.send resource names:%v failed: %v", DefaultSecretResourceNames, err2)
					}
				}
				continue
			}

			//if err == io.EOF {
			//	continue
			//}

			logger.Debugf("[sds channel] sds recv resp: %v", resp)

			for _, res := range resp.Resources {
				if res.GetTypeUrl() == resource.SecretType {
					secret := &tls.Secret{}
					if err := ptypes.UnmarshalAny(res, secret); err != nil {
						logger.Errorf("[sds channel] fail to extract secret name: %v", err)
						continue
					}
					sds.updateChan <- secret
				}
			}
			sds.AckResponse(resp)
		}
	}()

	go func() {
		for {
			select {
			case <-sds.stopChan:
				sds.Stop()
				return
			case secret, ok := <-sds.updateChan:
				if !ok {
					continue
				}
				for name, listener := range sds.listeners {
					err := listener(secret)
					if err != nil {
						logger.Errorf("[sds channel] sds.call.listener %s error:%v", name, err)
					}
				}
			}
		}

	}()

	<-sds.stopChan
}

func (sds *SdsClientChannel) reconnect() error {
	select {
	case <-time.After(1 * time.Second):
		logger.Infof("[sds channel] dealy 1 seconds to reconnect sds server")
	}
	logger.Info("[sds channel] reconnect sds server now")
	sds.closeConnection()
	newConn, err := grpc.Dial(
		sds.udsPath,
		grpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("[sds][reconnect] dial grpc server failed: %w", err)
	}

	sds.conn = newConn
	sds.secretDiscoveryClient = v3secret.NewSecretDiscoveryServiceClient(newConn)
	ctx, cancel := context.WithCancel(context.Background())
	sds.cancel = cancel
	streamSecretsClient, err := sds.secretDiscoveryClient.StreamSecrets(ctx)
	if err != nil {
		return fmt.Errorf("[sds][reconnect] get sds stream secret fail: %w", err)
	}
	sds.streamSecretsClient = streamSecretsClient
	logger.Info("[sds channel] reconnect sds server end")
	return nil
}

func (sds *SdsClientChannel) Send(names []string) error {
	request := &v3discovery.DiscoveryRequest{
		VersionInfo:   "",
		ResourceNames: names,
		TypeUrl:       resource.SecretType,
		ResponseNonce: "",
		ErrorDetail:   nil,
		Node:          sds.node,
	}
	logger.Infof("[sds channel] sds.send request = %v ", request)
	return sds.streamSecretsClient.Send(request)
}

func (sds *SdsClientChannel) InitSds() error {
	return sds.Send(DefaultSecretResourceNames)
}

func (sds *SdsClientChannel) AddListener(listener SecretUpdateListener, key string) {
	sds.listenerMutex.Lock()
	defer sds.listenerMutex.Unlock()
	sds.listeners[key] = listener
}

func (sds *SdsClientChannel) RemoveListener(key string) {
	sds.listenerMutex.Lock()
	defer sds.listenerMutex.Unlock()

	delete(sds.listeners, key)
}

func (sds *SdsClientChannel) AckResponse(resp interface{}) {
	xdsresp, ok := resp.(*v3discovery.DiscoveryResponse)

	if !ok {
		return
	}
	logger.Infof("[sds channel]  sds.send ack respoonse = %v ", xdsresp)
	if err := sds.ackResponse(xdsresp); err != nil {
		logger.Errorf("[sds channel] sds.send ack response  fail: %v", err)
	}

}

func (sds *SdsClientChannel) ackResponse(resp *v3discovery.DiscoveryResponse) error {
	secretNames := make([]string, 0)
	for _, resource := range resp.Resources {
		if resource == nil {
			continue
		}
		secret := &tls.Secret{}
		if err := ptypes.UnmarshalAny(resource, secret); err != nil {
			logger.Errorf("[sds channel] fail to extract secret name: %v", err)
			continue
		}
		secretNames = append(secretNames, secret.GetName())
	}

	req := &v3discovery.DiscoveryRequest{
		VersionInfo:   resp.VersionInfo,
		ResourceNames: secretNames,
		TypeUrl:       resp.TypeUrl,
		ResponseNonce: resp.Nonce,
		ErrorDetail:   nil,
		Node:          sds.node,
	}

	logger.Infof("[sds channel] sds.send a ack request to server: %v", req)
	return sds.streamSecretsClient.Send(req)
}

func (sds *SdsClientChannel) closeConnection() {
	sds.cancel()
	if sds.conn != nil {
		sds.conn.Close()
		sds.conn = nil
	}
}

func (sds *SdsClientChannel) Stop() {
	if runningStatus := sds.runningStatus.Load(); runningStatus {
		// make sure stop once
		sds.runningStatus.Store(false)
		logger.Infof("[sds channel] Stop now...")
		sds.closeConnection()
		close(sds.updateChan)
	}
}