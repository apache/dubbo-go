// Copyright 2016-2019 Yincheng Fang, Alex Stocks
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dubbo

import (
	"context"
	"reflect"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/getty"
	log "github.com/AlexStocks/log4go"
	"github.com/dubbogo/hessian2"
	perrors "github.com/pkg/errors"
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
	errTooManySessions = perrors.New("too many sessions")
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
		log.Error("client.getClientSession(session{%s}) = error{%v}",
			session.Stat(), perrors.WithStack(err))
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
		return perrors.WithStack(err)
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

	twoway := true
	// not twoway
	if p.Header.Type&hessian.PackageRequest_TwoWay == 0x00 {
		twoway = false
		h.reply(session, p, hessian.PackageResponse)
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
	if !twoway {
		return
	}
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
				log.Error("callService panic: %#v", perrors.New(err))
				req.Body = perrors.New(err)
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
		req.Body = perrors.New("service not found")
		return
	}
	svc := svcIf.(*common.Service)
	method := svc.Method()[req.Service.Method]
	if method == nil {
		log.Error("method not found!")
		req.Header.ResponseStatus = hessian.Response_SERVICE_NOT_FOUND
		req.Body = perrors.New("method not found")
		return
	}

	in := []reflect.Value{svc.Rcvr()}
	if method.CtxType() != nil {
		in = append(in, method.SuiteContext(ctx))
	}

	// prepare argv
	argv := req.Body.(map[string]interface{})["args"]
	if (len(method.ArgsType()) == 1 || len(method.ArgsType()) == 2 && method.ReplyType() == nil) && method.ArgsType()[0].String() == "[]interface {}" {
		in = append(in, reflect.ValueOf(argv))
	} else {
		for i := 0; i < len(argv.([]interface{})); i++ {
			in = append(in, reflect.ValueOf(argv.([]interface{})[i]))
		}
	}

	// prepare replyv
	var replyv reflect.Value
	if method.ReplyType() == nil {
		replyv = reflect.New(method.ArgsType()[len(method.ArgsType())-1].Elem())
		in = append(in, replyv)
	}

	returnValues := method.Method().Func.Call(in)

	var retErr interface{}
	if len(returnValues) == 1 {
		retErr = returnValues[0].Interface()
	} else {
		replyv = returnValues[0]
		retErr = returnValues[1].Interface()
	}
	if retErr != nil {
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
		log.Error("WritePkg error: %#v, %#v", perrors.WithStack(err), req.Header)
	}
}
