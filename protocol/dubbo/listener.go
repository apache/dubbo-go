package dubbo

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/getty"
	log "github.com/AlexStocks/log4go"
	"github.com/dubbogo/hessian2"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	"github.com/dubbo/go-for-apache-dubbo/protocol/invocation"
)

// todo: WritePkg_Timeout will entry *.yml
const WritePkg_Timeout = 5 * time.Second

var (
	errTooManySessions = jerrors.New("too many sessions")
)

type rpcSession struct {
	session getty.Session
	reqNum  int32
}

////////////////////////////////////////////
// RpcClientHandler
////////////////////////////////////////////

type RpcClientHandler struct {
	conn *gettyRPCClient
}

func NewRpcClientHandler(client *gettyRPCClient) *RpcClientHandler {
	return &RpcClientHandler{conn: client}
}

func (h *RpcClientHandler) OnOpen(session getty.Session) error {
	h.conn.addSession(session)
	return nil
}

func (h *RpcClientHandler) OnError(session getty.Session, err error) {
	log.Info("session{%s} got error{%v}, will be closed.", session.Stat(), err)
	h.conn.removeSession(session)
}

func (h *RpcClientHandler) OnClose(session getty.Session) {
	log.Info("session{%s} is closing......", session.Stat())
	h.conn.removeSession(session)
}

func (h *RpcClientHandler) OnMessage(session getty.Session, pkg interface{}) {
	p, ok := pkg.(*DubboPackage)
	if !ok {
		log.Error("illegal package")
		return
	}

	if p.Header.Type&hessian.PackageHeartbeat != 0x00 {
		log.Debug("get rpc heartbeat response{header: %#v, body: %#v}", p.Header, p.Body)
		return
	}
	log.Debug("get rpc response{header: %#v, body: %#v}", p.Header, p.Body)

	h.conn.updateSession(session)

	pendingResponse := h.conn.pool.rpcClient.removePendingResponse(SequenceType(p.Header.ID))
	if pendingResponse == nil {
		return
	}

	if p.Err != nil {
		pendingResponse.err = p.Err
	}

	if pendingResponse.callback == nil {
		pendingResponse.done <- struct{}{}
	} else {
		pendingResponse.callback(pendingResponse.GetCallResponse())
	}
}

func (h *RpcClientHandler) OnCron(session getty.Session) {
	rpcSession, err := h.conn.getClientRpcSession(session)
	if err != nil {
		log.Error("client.getClientSession(session{%s}) = error{%s}",
			session.Stat(), jerrors.ErrorStack(err))
		return
	}
	if h.conn.pool.rpcClient.conf.sessionTimeout.Nanoseconds() < time.Since(session.GetActive()).Nanoseconds() {
		log.Warn("session{%s} timeout{%s}, reqNum{%d}",
			session.Stat(), time.Since(session.GetActive()).String(), rpcSession.reqNum)
		h.conn.removeSession(session) // -> h.conn.close() -> h.conn.pool.remove(h.conn)
		return
	}

	h.conn.pool.rpcClient.heartbeat(session)
}

////////////////////////////////////////////
// RpcServerHandler
////////////////////////////////////////////

type RpcServerHandler struct {
	exporter       protocol.Exporter
	maxSessionNum  int
	sessionTimeout time.Duration
	sessionMap     map[getty.Session]*rpcSession
	rwlock         sync.RWMutex
}

func NewRpcServerHandler(exporter protocol.Exporter, maxSessionNum int, sessionTimeout time.Duration) *RpcServerHandler {
	return &RpcServerHandler{
		exporter:       exporter,
		maxSessionNum:  maxSessionNum,
		sessionTimeout: sessionTimeout,
		sessionMap:     make(map[getty.Session]*rpcSession),
	}
}

func (h *RpcServerHandler) OnOpen(session getty.Session) error {
	var err error
	h.rwlock.RLock()
	if h.maxSessionNum <= len(h.sessionMap) {
		err = errTooManySessions
	}
	h.rwlock.RUnlock()
	if err != nil {
		return jerrors.Trace(err)
	}

	log.Info("got session:%s", session.Stat())
	h.rwlock.Lock()
	h.sessionMap[session] = &rpcSession{session: session}
	h.rwlock.Unlock()
	return nil
}

func (h *RpcServerHandler) OnError(session getty.Session, err error) {
	log.Info("session{%s} got error{%v}, will be closed.", session.Stat(), err)
	h.rwlock.Lock()
	delete(h.sessionMap, session)
	h.rwlock.Unlock()
}

func (h *RpcServerHandler) OnClose(session getty.Session) {
	log.Info("session{%s} is closing......", session.Stat())
	h.rwlock.Lock()
	delete(h.sessionMap, session)
	h.rwlock.Unlock()
}

func (h *RpcServerHandler) OnMessage(session getty.Session, pkg interface{}) {
	h.rwlock.Lock()
	if _, ok := h.sessionMap[session]; ok {
		h.sessionMap[session].reqNum++
	}
	h.rwlock.Unlock()

	p, ok := pkg.(*DubboPackage)
	if !ok {
		log.Error("illegal packge{%#v}", pkg)
		return
	}
	p.Header.ResponseStatus = hessian.Response_OK

	// heartbeat
	if p.Header.Type&hessian.PackageHeartbeat != 0x00 {
		log.Debug("get rpc heartbeat request{header: %#v, service: %#v, body: %#v}", p.Header, p.Service, p.Body)
		h.reply(session, p, hessian.PackageHeartbeat)
		return
	}

	// not twoway
	if p.Header.Type&hessian.PackageRequest_TwoWay == 0x00 {
		h.reply(session, p, hessian.PackageResponse)
		h.callService(p, nil)
		return
	}

	invoker := h.exporter.GetInvoker()
	if invoker != nil {
		result := invoker.Invoke(invocation.NewRPCInvocationForProvider(p.Service.Method, p.Body.(map[string]interface{})["args"].([]interface{}), map[string]string{
			constant.PATH_KEY: p.Service.Path,
			//attachments[constant.GROUP_KEY] = url.GetParam(constant.GROUP_KEY, "")
			constant.INTERFACE_KEY: p.Service.Interface,
			constant.VERSION_KEY:   p.Service.Version,
		}))
		if err := result.Error(); err != nil {
			p.Header.ResponseStatus = hessian.Response_SERVER_ERROR
			p.Body = err
			h.reply(session, p, hessian.PackageResponse)
			return
		}
		if res := result.Result(); res != nil {
			p.Header.ResponseStatus = hessian.Response_OK
			p.Body = res
			h.reply(session, p, hessian.PackageResponse)
			return
		}
	}

	h.callService(p, nil)
	h.reply(session, p, hessian.PackageResponse)
}

func (h *RpcServerHandler) OnCron(session getty.Session) {
	var (
		flag   bool
		active time.Time
	)

	h.rwlock.RLock()
	if _, ok := h.sessionMap[session]; ok {
		active = session.GetActive()
		if h.sessionTimeout.Nanoseconds() < time.Since(active).Nanoseconds() {
			flag = true
			log.Warn("session{%s} timeout{%s}, reqNum{%d}",
				session.Stat(), time.Since(active).String(), h.sessionMap[session].reqNum)
		}
	}
	h.rwlock.RUnlock()

	if flag {
		h.rwlock.Lock()
		delete(h.sessionMap, session)
		h.rwlock.Unlock()
		session.Close()
	}
}

func (h *RpcServerHandler) callService(req *DubboPackage, ctx context.Context) {

	defer func() {
		if e := recover(); e != nil {
			req.Header.ResponseStatus = hessian.Response_BAD_REQUEST
			if err, ok := e.(error); ok {
				log.Error("callService panic: %#v", err)
				req.Body = e.(error)
			} else if err, ok := e.(string); ok {
				log.Error("callService panic: %#v", jerrors.New(err))
				req.Body = jerrors.New(err)
			} else {
				log.Error("callService panic: %#v", e)
				req.Body = e
			}
		}
	}()

	svcIf := req.Body.(map[string]interface{})["service"]
	if svcIf == nil {
		log.Error("service not found!")
		req.Header.ResponseStatus = hessian.Response_SERVICE_NOT_FOUND
		req.Body = errors.New("service not found")
		return
	}
	svc := svcIf.(*common.Service)
	method := svc.Method()[req.Service.Method]
	if method == nil {
		log.Error("method not found!")
		req.Header.ResponseStatus = hessian.Response_SERVICE_NOT_FOUND
		req.Body = errors.New("method not found")
		return
	}

	// prepare argv
	var argv reflect.Value
	argIsValue := false // if true, need to indirect before calling.
	if method.ArgType().Kind() == reflect.Ptr {
		argv = reflect.New(method.ArgType().Elem())
	} else {
		argv = reflect.New(method.ArgType())
		argIsValue = true
	}
	argvTmp := argv.Interface()
	argvTmp = req.Body.(map[string]interface{})["args"] // type is []interface
	if argIsValue {
		argv = argv.Elem()
	}

	// prepare replyv
	replyv := reflect.New(method.ReplyType().Elem())
	var returnValues []reflect.Value
	if method.CtxType() == nil {
		returnValues = method.Method().Func.Call([]reflect.Value{svc.Rcvr(), reflect.ValueOf(argvTmp), reflect.ValueOf(replyv.Interface())})
	} else {
		if contextv := reflect.ValueOf(ctx); contextv.IsValid() {
			returnValues = method.Method().Func.Call([]reflect.Value{svc.Rcvr(), contextv, reflect.ValueOf(argvTmp), reflect.ValueOf(replyv.Interface())})
		} else {
			returnValues = method.Method().Func.Call([]reflect.Value{svc.Rcvr(), reflect.Zero(method.CtxType()), reflect.ValueOf(argvTmp), reflect.ValueOf(replyv.Interface())})
		}
	}

	// The return value for the method is an error.
	if retErr := returnValues[0].Interface(); retErr != nil {
		req.Header.ResponseStatus = hessian.Response_SERVER_ERROR
		req.Body = retErr.(error)
	} else {
		req.Body = replyv.Interface()
	}
}

func (h *RpcServerHandler) reply(session getty.Session, req *DubboPackage, tp hessian.PackageType) {
	resp := &DubboPackage{
		Header: hessian.DubboHeader{
			SerialID:       req.Header.SerialID,
			Type:           tp,
			ID:             req.Header.ID,
			ResponseStatus: req.Header.ResponseStatus,
		},
	}

	if req.Header.Type&hessian.PackageRequest != 0x00 {
		resp.Body = req.Body
	} else {
		resp.Body = nil
	}

	if err := session.WritePkg(resp, WritePkg_Timeout); err != nil {
		log.Error("WritePkg error: %#v, %#v", jerrors.Trace(err), req.Header)
	}
}
