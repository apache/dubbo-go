package grpc_new

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	hessian2 "github.com/apache/dubbo-go-hessian2"
	"github.com/dubbogo/gost/log/logger"
	"github.com/pkg/errors"
	"reflect"
	"sync"
)

var errNoReply = errors.New("request need @response")

type GrpcNewInvoker struct {
	protocol.BaseInvoker
	quitOnce    sync.Once
	clientGuard *sync.RWMutex
	client      *Client
}

func (gni *GrpcNewInvoker) setClient(client *Client) {
	gni.clientGuard.Lock()
	defer gni.clientGuard.Unlock()

	gni.client = client
}

func (gni *GrpcNewInvoker) getClient() *Client {
	gni.clientGuard.RLock()
	defer gni.clientGuard.RUnlock()

	return gni.client
}

// Invoke is used to call service method by invocation
func (gni *GrpcNewInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	var result protocol.RPCResult

	if !gni.BaseInvoker.IsAvailable() {
		// Generally, the case will not happen, because the invoker has been removed
		// from the invoker list before destroy,so no new request will enter the destroyed invoker
		logger.Warnf("this grpcInvoker is destroyed")
		result.Err = protocol.ErrDestroyedInvoker
		return &result
	}

	gni.clientGuard.RLock()
	defer gni.clientGuard.RUnlock()

	if gni.client == nil {
		result.Err = protocol.ErrClientClosed
		return &result
	}

	if !gni.BaseInvoker.IsAvailable() {
		// Generally, the case will not happen, because the invoker has been removed
		// from the invoker list before destroy,so no new request will enter the destroyed invoker
		logger.Warnf("this grpcInvoker is destroying")
		result.Err = protocol.ErrDestroyedInvoker
		return &result
	}

	if invocation.Reply() == nil {
		result.Err = errNoReply
	}

	var in []reflect.Value
	in = append(in, reflect.ValueOf(ctx))
	in = append(in, invocation.ParameterValues()...)

	methodName := invocation.MethodName()
	method := gni.client.invoker.MethodByName(methodName)
	res := method.Call(in)

	result.Rest = res[0]
	// check err
	if !res[1].IsNil() {
		result.Err = res[1].Interface().(error)
	} else {
		// todo figure this out
		_ = hessian2.ReflectResponse(res[0], invocation.Reply())
	}

	return &result
}

// todo: add state to client
// IsAvailable get available status
func (gni *GrpcNewInvoker) IsAvailable() bool {
	client := gni.getClient()
	if client != nil {
		return gni.BaseInvoker.IsAvailable()
	}

	return false
}

// todo: add state
// IsDestroyed get destroyed status
func (gni *GrpcNewInvoker) IsDestroyed() bool {
	client := gni.getClient()
	if client != nil {
		return gni.BaseInvoker.IsDestroyed()
	}

	return false
}

// Destroy will destroy gRPC's invoker and client, so it is only called once
func (gni *GrpcNewInvoker) Destroy() {
	gni.quitOnce.Do(func() {
		gni.BaseInvoker.Destroy()
		client := gni.getClient()
		if client != nil {
			gni.setClient(nil)
			// todo:
			client.CloseIdleConnections()
		}
	})
}

func NewGrpcNewInvoker(url *common.URL, client *Client) *GrpcNewInvoker {
	return &GrpcNewInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		quitOnce:    sync.Once{},
		clientGuard: &sync.RWMutex{},
		client:      client,
	}
}
