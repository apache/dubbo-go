package channel

import (
	"context"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	v3configcore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	tls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	v3discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	v3secret "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"io"
	"sync"
	"time"
)

var (
	DefaultSecretResourceNames = []string{"default", "ROOTCA"}
)

type SecretUpdateListener func(*tls.Secret) error

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
}

func NewSdsClientChannel(stopChan chan struct{}, sdsUdsPath string, node *v3configcore.Node) (*SdsClientChannel, error) {
	udsPath := "unix:" + sdsUdsPath
	conn, err := grpc.Dial(
		udsPath,
		grpc.WithInsecure(),
	)
	if err != nil {
		logger.Errorf("sds.subscribe.stream [sds][subscribe] dial grpc server failed %v", err)
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
		logger.Errorf("sds.subscribe.stream [sds][subscribe] get sds stream secret fail %v", err)
		conn.Close()
		return nil, err
	}
	go sdsStreamClient.startListening()
	return sdsStreamClient, nil
}

func (sds *SdsClientChannel) startListening() {
	go func() {
		// need to subscribe all resources
		//if err := sds.Send([]string{"default", "ROOTCA"}); err != nil {
		//	logger.Errorf("sds.send.error [sds][send] failed: %v", err)
		//}
		for {
			select {
			case <-sds.stopChan:
				return
			default:
				resp, err := sds.streamSecretsClient.Recv()

				if err != nil && err != io.EOF {
					logger.Errorf("sds.recv.error [sds][recv] error receiving secrets: %v", err)
					if err := sds.reconnect(); err != nil {
						logger.Errorf("sds.reconnect.error [sds][reconnect] failed to reconnect: %v", err)
						continue
					} else {
						// need to subscribe all resources again
						if err2 := sds.Send(DefaultSecretResourceNames); err2 != nil {
							logger.Errorf("sds.send.error [sds][send] resource names:%v failed: %v", DefaultSecretResourceNames, err2)
						}
					}
					continue
				}

				if err == io.EOF {
					continue
				}

				logger.Infof("sds recv resp %v", resp)

				for _, res := range resp.Resources {
					if res.GetTypeUrl() == resource.SecretType {
						secret := &tls.Secret{}
						if err := ptypes.UnmarshalAny(res, secret); err != nil {
							logger.Errorf("fail to extract secret name: %v", err)
							continue
						}
						sds.updateChan <- secret
					}
				}
				sds.AckResponse(resp)
			}
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
						logger.Errorf("sds call listener %s error:%v", name, err)
					}
				}
			}
		}

	}()

	<-sds.stopChan
}

func (sds *SdsClientChannel) reconnect() error {
	sds.closeConnection()

	select {
	case <-sds.stopChan:
		return fmt.Errorf("stop chan stoped")
	case <-time.After(2 * time.Second):
		logger.Infof("dealy 2 seconds to reconnect sds server")
	}

	newConn, err := grpc.Dial(
		sds.udsPath,
		grpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("[sds][reconnect] dial grpc server failed: %w", err)
	}

	sds.conn = newConn
	sds.secretDiscoveryClient = v3secret.NewSecretDiscoveryServiceClient(newConn)
	streamSecretsClient, err := sds.secretDiscoveryClient.StreamSecrets(context.Background())
	if err != nil {
		return fmt.Errorf("[sds][reconnect] get sds stream secret fail: %w", err)
	}
	sds.streamSecretsClient = streamSecretsClient

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
	logger.Infof("sds send request = %v ", request)
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
	logger.Infof("sds send ack respoonse = %v ", xdsresp)
	if err := sds.ackResponse(xdsresp); err != nil {
		logger.Errorf("sds send ack response  fail: %v", err)
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
			logger.Errorf("fail to extract secret name: %v", err)
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

	logger.Infof("send a ack request to server: %v", req)
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
	sds.closeConnection()
	close(sds.updateChan)
}
