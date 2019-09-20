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
	"fmt"
	"net/url"
	"sync"
	"time"
)

import (
	"github.com/apache/dubbo-go-hessian2"
	"github.com/dubbogo/getty"
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
	logger.Infof("session{%s} got error{%v}, will be closed.", session.Stat(), err)
	h.conn.removeSession(session)
}

func (h *RpcClientHandler) OnClose(session getty.Session) {
	logger.Infof("session{%s} is closing......", session.Stat())
	h.conn.removeSession(session)
}

func (h *RpcClientHandler) OnMessage(session getty.Session, pkg interface{}) {
	p, ok := pkg.(*DubboPackage)
	if !ok {
		logger.Errorf("illegal package")
		return
	}

	if p.Header.Type&hessian.PackageHeartbeat != 0x00 {
		logger.Debugf("get rpc heartbeat response{header: %#v, body: %#v}", p.Header, p.Body)
		if p.Err != nil {
			logger.Errorf("rpc heartbeat response{error: %#v}", p.Err)
		}
		h.conn.pool.rpcClient.removePendingResponse(SequenceType(p.Header.ID))
		return
	}
	logger.Debugf("get rpc response{header: %#v, body: %#v}", p.Header, p.Body)

	h.conn.updateSession(session)

	pendingResponse := h.conn.pool.rpcClient.removePendingResponse(SequenceType(p.Header.ID))
	if pendingResponse == nil {
		return
	}

	if p.Err != nil {
		pendingResponse.err = p.Err
	}

	pendingResponse.response.atta = p.Body.(*Response).atta

	if pendingResponse.callback == nil {
		pendingResponse.done <- struct{}{}
	} else {
		pendingResponse.callback(pendingResponse.GetCallResponse())
	}
}

func (h *RpcClientHandler) OnCron(session getty.Session) {
	rpcSession, err := h.conn.getClientRpcSession(session)
	if err != nil {
		logger.Errorf("client.getClientSession(session{%s}) = error{%v}",
			session.Stat(), perrors.WithStack(err))
		return
	}
	if h.conn.pool.rpcClient.conf.sessionTimeout.Nanoseconds() < time.Since(session.GetActive()).Nanoseconds() {
		logger.Warnf("session{%s} timeout{%s}, reqNum{%d}",
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
	maxSessionNum  int
	sessionTimeout time.Duration
	sessionMap     map[getty.Session]*rpcSession
	rwlock         sync.RWMutex
}

func NewRpcServerHandler(maxSessionNum int, sessionTimeout time.Duration) *RpcServerHandler {
	return &RpcServerHandler{
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

	logger.Infof("got session:%s", session.Stat())
	h.rwlock.Lock()
	h.sessionMap[session] = &rpcSession{session: session}
	h.rwlock.Unlock()
	return nil
}

func (h *RpcServerHandler) OnError(session getty.Session, err error) {
	logger.Infof("session{%s} got error{%v}, will be closed.", session.Stat(), err)
	h.rwlock.Lock()
	delete(h.sessionMap, session)
	h.rwlock.Unlock()
}

func (h *RpcServerHandler) OnClose(session getty.Session) {
	logger.Infof("session{%s} is closing......", session.Stat())
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
		logger.Errorf("illegal package{%#v}", pkg)
		return
	}
	p.Header.ResponseStatus = hessian.Response_OK

	// heartbeat
	if p.Header.Type&hessian.PackageHeartbeat != 0x00 {
		logger.Debugf("get rpc heartbeat request{header: %#v, service: %#v, body: %#v}", p.Header, p.Service, p.Body)
		h.reply(session, p, hessian.PackageHeartbeat)
		return
	}

	twoway := true
	// not twoway
	if p.Header.Type&hessian.PackageRequest_TwoWay == 0x00 {
		twoway = false
	}

	defer func() {
		if e := recover(); e != nil {
			p.Header.ResponseStatus = hessian.Response_SERVER_ERROR
			if err, ok := e.(error); ok {
				logger.Errorf("OnMessage panic: %+v", perrors.WithStack(err))
				p.Body = perrors.WithStack(err)
			} else if err, ok := e.(string); ok {
				logger.Errorf("OnMessage panic: %+v", perrors.New(err))
				p.Body = perrors.New(err)
			} else {
				logger.Errorf("OnMessage panic: %+v, this is impossible.", e)
				p.Body = e
			}

			if !twoway {
				return
			}
			h.reply(session, p, hessian.PackageResponse)
		}

	}()

	u := common.NewURLWithOptions(common.WithPath(p.Service.Path), common.WithParams(url.Values{}),
		common.WithParamsValue(constant.GROUP_KEY, p.Service.Group),
		common.WithParamsValue(constant.INTERFACE_KEY, p.Service.Interface),
		common.WithParamsValue(constant.VERSION_KEY, p.Service.Version))
	exporter, _ := dubboProtocol.ExporterMap().Load(u.ServiceKey())
	if exporter == nil {
		err := fmt.Errorf("don't have this exporter, key: %s", u.ServiceKey())
		logger.Errorf(err.Error())
		p.Header.ResponseStatus = hessian.Response_OK
		p.Body = err
		h.reply(session, p, hessian.PackageResponse)
		return
	}
	invoker := exporter.(protocol.Exporter).GetInvoker()
	if invoker != nil {
		result := invoker.Invoke(invocation.NewRPCInvocation(p.Service.Method, p.Body.(map[string]interface{})["args"].([]interface{}),
			p.Body.(map[string]interface{})["attachments"].(map[string]string)))
		if err := result.Error(); err != nil {
			p.Header.ResponseStatus = hessian.Response_OK
			p.Body = hessian.NewResponse(nil, err, result.Attachments())
		} else {
			res := result.Result()
			p.Header.ResponseStatus = hessian.Response_OK
			p.Body = hessian.NewResponse(res, nil, result.Attachments())
		}
	}

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
		logger.Errorf("WritePkg error: %#v, %#v", perrors.WithStack(err), req.Header)
	}
}
