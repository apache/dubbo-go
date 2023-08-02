package triple

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/dubbogo/gost/log/logger"
	"sync"
)

type TripleInvoker struct {
	protocol.BaseInvoker
	quitOnce      sync.Once
	clientGuard   *sync.RWMutex
	clientManager *clientManager
}

func (gni *TripleInvoker) setClientManager(cm *clientManager) {
	gni.clientGuard.Lock()
	defer gni.clientGuard.Unlock()

	gni.clientManager = cm
}

func (gni *TripleInvoker) getClientManager() *clientManager {
	gni.clientGuard.RLock()
	defer gni.clientGuard.RUnlock()

	return gni.clientManager
}

// Invoke is used to call client-side method.
func (ti *TripleInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	var result protocol.RPCResult

	if !ti.BaseInvoker.IsAvailable() {
		// Generally, the case will not happen, because the invoker has been removed
		// from the invoker list before destroy,so no new request will enter the destroyed invoker
		logger.Warnf("TripleInvoker is destroyed")
		result.Err = protocol.ErrDestroyedInvoker
		return &result
	}

	ti.clientGuard.RLock()
	defer ti.clientGuard.RUnlock()

	if ti.clientManager == nil {
		result.Err = protocol.ErrClientClosed
		return &result
	}
	consumeTypeRaw, ok := invocation.GetAttribute(constant.ConsumeTypeKey)
	if !ok {
		panic("")
	}
	consumeType, ok := consumeTypeRaw.(string)
	if !ok {
		panic("")
	}
	inRaw := invocation.ParameterRawValues()
	method := invocation.MethodName()
	switch consumeType {
	case constant.ConsumeUnary:
		if len(inRaw) != 2 {
			panic("")
		}
		if err := ti.clientManager.callUnary(ctx, method, inRaw[0], inRaw[1]); err != nil {
			result.Err = err
			return &result
		}
	case constant.ConsumeClientStream:
		stream, err := ti.clientManager.callClientStream(ctx, method)
		if err != nil {
			result.Err = err
			return &result
		}
		result.SetResult(stream)
		ReflectResponse(stream, invocation.Reply())
		return &result
	case constant.ConsumeServerStream:
		if len(inRaw) != 1 {
			panic("")
		}
		stream, err := ti.clientManager.callServerStream(ctx, method, inRaw[0])
		if err != nil {
			result.Err = err
			return &result
		}
		result.SetResult(stream)
		ReflectResponse(stream, invocation.Reply())
		return &result
	case constant.ConsumeBidiStream:
		stream, err := ti.clientManager.callBidiStream(ctx, method)
		if err != nil {
			result.Err = err
			return &result
		}
		result.SetResult(stream)
		ReflectResponse(stream, invocation.Reply())
		return &result
	default:
		panic("")
	}

	return &result
}

// IsAvailable get available status
func (gni *TripleInvoker) IsAvailable() bool {
	if gni.getClientManager() != nil {
		return gni.BaseInvoker.IsAvailable()
	}

	return false
}

// IsDestroyed get destroyed status
func (gni *TripleInvoker) IsDestroyed() bool {
	if gni.getClientManager() != nil {
		return gni.BaseInvoker.IsDestroyed()
	}

	return false
}

// Destroy will destroy Triple's invoker and client, so it is only called once
func (ti *TripleInvoker) Destroy() {
	ti.quitOnce.Do(func() {
		ti.BaseInvoker.Destroy()
		if cm := ti.getClientManager(); cm != nil {
			ti.setClientManager(nil)
			// todo:// find a better way to destroy these resources
			cm.close()
		}
	})
}

func NewTripleInvoker(url *common.URL) (*TripleInvoker, error) {
	cm, err := newClientManager(url)
	if err != nil {
		return nil, err
	}
	return &TripleInvoker{
		BaseInvoker:   *protocol.NewBaseInvoker(url),
		quitOnce:      sync.Once{},
		clientGuard:   &sync.RWMutex{},
		clientManager: cm,
	}, nil
}
