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

package dubbo

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-go/protocol/dubbo/impl"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/apache/dubbo-go-hessian2"
	"github.com/dubbogo/getty"
	"github.com/opentracing/opentracing-go"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

// todo: WritePkg_Timeout will entry *.yml
const (
	// WritePkg_Timeout ...
	WritePkg_Timeout = 5 * time.Second
)

var (
	errTooManySessions = perrors.New("too many sessions")
)

type rpcSession struct {
	session getty.Session
	reqNum  int32
}

func (s *rpcSession) AddReqNum(num int32) {
	atomic.AddInt32(&s.reqNum, num)
}

func (s *rpcSession) GetReqNum() int32 {
	return atomic.LoadInt32(&s.reqNum)
}

// //////////////////////////////////////////
// RpcClientHandler
// //////////////////////////////////////////

// RpcClientHandler ...
type RpcClientHandler struct {
	conn *gettyRPCClient
}

// NewRpcClientHandler ...
func NewRpcClientHandler(client *gettyRPCClient) *RpcClientHandler {
	return &RpcClientHandler{conn: client}
}

// OnOpen ...
func (h *RpcClientHandler) OnOpen(session getty.Session) error {
	h.conn.addSession(session)
	return nil
}

// OnError ...
func (h *RpcClientHandler) OnError(session getty.Session, err error) {
	logger.Infof("session{%s} got error{%v}, will be closed.", session.Stat(), err)
	h.conn.removeSession(session)
}

// OnClose ...
func (h *RpcClientHandler) OnClose(session getty.Session) {
	logger.Infof("session{%s} is closing......", session.Stat())
	h.conn.removeSession(session)
}

// OnMessage ...
func (h *RpcClientHandler) OnMessage(session getty.Session, pkg interface{}) {
	p, ok := pkg.(*impl.DubboPackage)
	if !ok {
		logger.Errorf("illegal package")
		return
	}

	if p.Header.Type&impl.PackageHeartbeat != 0x00 {
		logger.Debugf("get rpc heartbeat response{header: %#v, body: %#v}", p.Header, p.Body)
		if p.Err != nil {
			logger.Errorf("rpc heartbeat response{error: %#v}", p.Err)
		}
		return
	}
	logger.Debugf("get rpc response{header: %#v, body: %#v}", p.Header, p.Body)

	h.conn.updateSession(session)

	pendingResponse := h.conn.pool.rpcClient.removePendingResponse(impl.SequenceType(p.Header.ID))
	if pendingResponse == nil {
		logger.Errorf("failed to get pending response context for response package %s", *p)
		return
	}

	if p.Err != nil {
		pendingResponse.err = p.Err
	}

	pendingResponse.response.atta = p.Body.(*Response).atta

	if pendingResponse.callback == nil {
		pendingResponse.done <- struct{}{}
	} else {
		logger.Info("proxy service callback")
		pendingResponse.callback(pendingResponse.GetCallResponse())
	}
}

// OnCron ...
func (h *RpcClientHandler) OnCron(session getty.Session) {
	rpcSession, err := h.conn.getClientRpcSession(session)
	if err != nil {
		logger.Errorf("client.getClientSession(session{%s}) = error{%v}",
			session.Stat(), perrors.WithStack(err))
		return
	}
	if h.conn.pool.rpcClient.conf.SessionTimeoutD.Nanoseconds() < time.Since(session.GetActive()).Nanoseconds() {
		logger.Warnf("session{%s} timeout{%s}, reqNum{%d}",
			session.Stat(), time.Since(session.GetActive()).String(), rpcSession.reqNum)
		h.conn.removeSession(session) // -> h.conn.close() -> h.conn.pool.remove(h.conn)
		return
	}

	h.conn.pool.rpcClient.heartbeat(session)
}

// //////////////////////////////////////////
// RpcServerHandler
// //////////////////////////////////////////

// RpcServerHandler ...
type RpcServerHandler struct {
	maxSessionNum  int
	sessionTimeout time.Duration
	sessionMap     map[getty.Session]*rpcSession
	rwlock         sync.RWMutex
}

// NewRpcServerHandler ...
func NewRpcServerHandler(maxSessionNum int, sessionTimeout time.Duration) *RpcServerHandler {
	return &RpcServerHandler{
		maxSessionNum:  maxSessionNum,
		sessionTimeout: sessionTimeout,
		sessionMap:     make(map[getty.Session]*rpcSession),
	}
}

// OnOpen ...
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

	logger.Infof("got session:%s", session.Stat())
	h.rwlock.Lock()
	h.sessionMap[session] = &rpcSession{session: session}
	h.rwlock.Unlock()
	return nil
}

// OnError ...
func (h *RpcServerHandler) OnError(session getty.Session, err error) {
	logger.Infof("session{%s} got error{%v}, will be closed.", session.Stat(), err)
	h.rwlock.Lock()
	delete(h.sessionMap, session)
	h.rwlock.Unlock()
}

// OnClose ...
func (h *RpcServerHandler) OnClose(session getty.Session) {
	logger.Infof("session{%s} is closing......", session.Stat())
	h.rwlock.Lock()
	delete(h.sessionMap, session)
	h.rwlock.Unlock()
}

// OnMessage ...
func (h *RpcServerHandler) OnMessage(session getty.Session, pkg interface{}) {
	h.rwlock.Lock()
	if _, ok := h.sessionMap[session]; ok {
		h.sessionMap[session].reqNum++
	}
	h.rwlock.Unlock()

	p, ok := pkg.(*impl.DubboPackage)
	if !ok {
		logger.Errorf("illegal package{%#v}", pkg)
		return
	}
	p.SetResponseStatus(hessian.Response_OK)
	//p.Header.ResponseStatus = hessian.Response_OK

	// heartbeat
	if p.GetHeader().Type&impl.PackageHeartbeat != 0x00 {
		logger.Debugf("get rpc heartbeat request{header: %#v, service: %#v, body: %#v}", p.GetHeader(), p.GetService(), p.GetBody())
		h.reply(session, p, impl.PackageHeartbeat)
		return
	}

	twoway := true
	// not twoway
	if p.GetHeader().Type&impl.PackageRequest_TwoWay == 0x00 {
		twoway = false
	}

	defer func() {
		if e := recover(); e != nil {
			p.SetResponseStatus(hessian.Response_SERVER_ERROR)
			if err, ok := e.(error); ok {
				logger.Errorf("OnMessage panic: %+v", perrors.WithStack(err))
				p.SetBody(perrors.WithStack(err))
			} else if err, ok := e.(string); ok {
				logger.Errorf("OnMessage panic: %+v", perrors.New(err))
				p.SetBody(perrors.New(err))
			} else {
				logger.Errorf("OnMessage panic: %+v, this is impossible.", e)
				p.SetBody(e)
			}

			if !twoway {
				return
			}
			h.reply(session, p, impl.PackageResponse)
		}

	}()

	u := common.NewURLWithOptions(common.WithPath(p.GetService().Path), common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GROUP_KEY, p.GetService().Group),
		common.WithParamsValue(constant.INTERFACE_KEY, p.GetService().Interface),
		common.WithParamsValue(constant.VERSION_KEY, p.GetService().Version))
	exporter, _ := dubboProtocol.ExporterMap().Load(u.ServiceKey())
	if exporter == nil {
		err := fmt.Errorf("don't have this exporter, key: %s", u.ServiceKey())
		logger.Errorf(err.Error())
		p.SetResponseStatus(impl.Response_OK)
		p.SetBody(err)
		h.reply(session, p, impl.PackageResponse)
		return
	}
	invoker := exporter.(protocol.Exporter).GetInvoker()
	if invoker != nil {
		attachments := p.GetBody().(map[string]interface{})["attachments"].(map[string]string)
		attachments[constant.LOCAL_ADDR] = session.LocalAddr()
		attachments[constant.REMOTE_ADDR] = session.RemoteAddr()

		args := p.GetBody().(map[string]interface{})["args"].([]interface{})
		inv := invocation.NewRPCInvocation(p.GetService().Method, args, attachments)

		ctx := rebuildCtx(inv)
		result := invoker.Invoke(ctx, inv)
		logger.Debugf("invoker result: %+v", result)
		if err := result.Error(); err != nil {
			p.SetResponseStatus(impl.Response_OK)
			p.SetBody(&impl.ResponsePayload{nil, err, result.Attachments()})
		} else {
			res := result.Result()
			p.SetResponseStatus(impl.Response_OK)
			p.SetBody(&impl.ResponsePayload{res, nil, result.Attachments()})
			//logger.Debugf("service return response %v", res)
		}
	}

	if !twoway {
		return
	}
	h.reply(session, p, impl.PackageResponse)
}

// OnCron ...
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
			logger.Warnf("session{%s} timeout{%s}, reqNum{%d}",
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

// rebuildCtx rebuild the context by attachment.
// Once we decided to transfer more context's key-value, we should change this.
// now we only support rebuild the tracing context
func rebuildCtx(inv *invocation.RPCInvocation) context.Context {
	ctx := context.Background()

	// actually, if user do not use any opentracing framework, the err will not be nil.
	spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.TextMap,
		opentracing.TextMapCarrier(inv.Attachments()))
	if err == nil {
		ctx = context.WithValue(ctx, constant.TRACING_REMOTE_SPAN_CTX, spanCtx)
	}
	return ctx
}

func (h *RpcServerHandler) reply(session getty.Session, req *impl.DubboPackage, tp impl.PackageType) {
	header := impl.DubboHeader{
		SerialID:       req.GetHeader().SerialID,
		Type:           tp,
		ID:             req.GetHeader().ID,
		BodyLen:        0,
		ResponseStatus: req.GetHeader().ResponseStatus,
	}
	resp := NewServerResponsePackage(header)
	if err := loadSerializer(resp); err != nil {
		logger.Errorf("reply error %v", err)
		return
	}

	if req.GetHeader().Type&impl.PackageRequest != 0x00 {
		resp.SetBody(req.GetBody())
	}

	if err := session.WritePkg(resp, WritePkg_Timeout); err != nil {
		logger.Errorf("WritePkg error: %#v, %#v", perrors.WithStack(err), req.GetHeader())
	}
}

// server side response package, just for serialization
func NewServerResponsePackage(header impl.DubboHeader) *impl.DubboPackage {
	return &impl.DubboPackage{
		Header: header,
		Body:   nil,
		Err:    nil,
		Codec:  impl.NewDubboCodec(nil),
	}
}
