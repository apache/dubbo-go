package dubbo3

import (
	"context"
	hessian2 "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
	invocation_impl "github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/remoting/dubbo3"
	"github.com/opentracing/opentracing-go"
	perrors "github.com/pkg/errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// ErrNoReply
	ErrNoReply = perrors.New("request need @response")
	// ErrDestroyedInvoker
	ErrDestroyedInvoker = perrors.New("request Destroyed invoker")
)

var (
	attachmentKey = []string{constant.INTERFACE_KEY, constant.GROUP_KEY, constant.TOKEN_KEY, constant.TIMEOUT_KEY,
		constant.VERSION_KEY}
)

// Dubbo3Invoker is implement of protocol.Invoker. A dubboInvoker refer to one service and ip.
type Dubbo3Invoker struct {
	protocol.BaseInvoker
	// the exchange layer, it is focus on network communication.
	client   *dubbo3.TripleClient
	quitOnce sync.Once
	// timeout for service(interface) level.
	timeout time.Duration
	// Used to record the number of requests. -1 represent this DubboInvoker is destroyed
	reqNum int64
}

// NewDubbo3Invoker constructor
func NewDubbo3Invoker(url *common.URL) *Dubbo3Invoker {
	requestTimeout := config.GetConsumerConfig().RequestTimeout
	requestTimeoutStr := url.GetParam(constant.TIMEOUT_KEY, config.GetConsumerConfig().Request_Timeout)
	if t, err := time.ParseDuration(requestTimeoutStr); err == nil {
		requestTimeout = t
	}

	client := dubbo3.NewTripleClient(url)

	return &Dubbo3Invoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		client:      client,
		reqNum:      0,
		timeout:     requestTimeout,
	}
}


// Invoke call remoting.
func (di *Dubbo3Invoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	var (
		result protocol.RPCResult
	)

	var in []reflect.Value
	in = append(in, reflect.ValueOf(context.Background()))
	// 这里invocation.ParameterValues()就是要传入的value
	in = append(in, invocation.ParameterValues()...)

	methodName := invocation.MethodName()
	method := di.client.Invoker.MethodByName(methodName)
	res := method.Call(in)

	result.Rest = res[0]
	// check err
	if !res[1].IsNil() {
		result.Err = res[1].Interface().(error)
	} else {
		_ = hessian2.ReflectResponse(res[0], invocation.Reply())
	}

	return &result
}



// get timeout including methodConfig
func (di *Dubbo3Invoker) getTimeout(invocation *invocation_impl.RPCInvocation) time.Duration {
	var timeout = di.GetUrl().GetParam(strings.Join([]string{constant.METHOD_KEYS, invocation.MethodName(), constant.TIMEOUT_KEY}, "."), "")
	if len(timeout) != 0 {
		if t, err := time.ParseDuration(timeout); err == nil {
			// config timeout into attachment
			invocation.SetAttachments(constant.TIMEOUT_KEY, strconv.Itoa(int(t.Milliseconds())))
			return t
		}
	}
	// set timeout into invocation at method level
	invocation.SetAttachments(constant.TIMEOUT_KEY, strconv.Itoa(int(di.timeout.Milliseconds())))
	return di.timeout
}

func (di *Dubbo3Invoker) IsAvailable() bool {
	return di.client.IsAvailable()
}

// Destroy destroy dubbo client invoker.
func (di *Dubbo3Invoker) Destroy() {
	di.quitOnce.Do(func() {
		for {
			if di.reqNum == 0 {
				di.reqNum = -1
				logger.Infof("dubboInvoker is destroyed,url:{%s}", di.GetUrl().Key())
				di.BaseInvoker.Destroy()
				if di.client != nil {
					di.client.Close()
					di.client = nil
				}
				break
			}
			logger.Warnf("DubboInvoker is to be destroyed, wait {%v} req end,url:{%s}", di.reqNum, di.GetUrl().Key())
			time.Sleep(1 * time.Second)
		}

	})
}

// Finally, I made the decision that I don't provide a general way to transfer the whole context
// because it could be misused. If the context contains to many key-value pairs, the performance will be much lower.
func (di *Dubbo3Invoker) appendCtx(ctx context.Context, inv *invocation_impl.RPCInvocation) {
	// inject opentracing ctx
	currentSpan := opentracing.SpanFromContext(ctx)
	if currentSpan != nil {
		err := injectTraceCtx(currentSpan, inv)
		if err != nil {
			logger.Errorf("Could not inject the span context into attachments: %v", err)
		}
	}
}
