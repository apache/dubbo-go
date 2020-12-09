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

package getty

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

import (
	"github.com/apache/dubbo-getty"
	hessian "github.com/apache/dubbo-go-hessian2"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/remoting"
)

// todo: WritePkg_Timeout will entry *.yml
const (
	// WritePkg_Timeout the timeout of write pkg
	WritePkg_Timeout = 5 * time.Second
)

var (
	errTooManySessions      = perrors.New("too many sessions")
	errHeartbeatReadTimeout = perrors.New("heartbeat read timeout")
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

// nolint
type RpcClientHandler struct {
	conn         *gettyRPCClient
	timeoutTimes int
}

// nolint
func NewRpcClientHandler(client *gettyRPCClient) *RpcClientHandler {
	return &RpcClientHandler{conn: client}
}

// OnOpen call the getty client session opened, add the session to getty client session list
func (h *RpcClientHandler) OnOpen(session getty.Session) error {
	h.conn.addSession(session)
	return nil
}

// OnError the getty client session has errored, so remove the session from the getty client session list
func (h *RpcClientHandler) OnError(session getty.Session, err error) {
	logger.Infof("session{%s} got error{%v}, will be closed.", session.Stat(), err)
	h.conn.removeSession(session)
}

// OnClose close the session, remove it from the getty session list
func (h *RpcClientHandler) OnClose(session getty.Session) {
	logger.Infof("session{%s} is closing......", session.Stat())
	h.conn.removeSession(session)
}

// OnMessage get response from getty server, and update the session to the getty client session list
func (h *RpcClientHandler) OnMessage(session getty.Session, pkg interface{}) {
	result, ok := pkg.(remoting.DecodeResult)
	if !ok {
		logger.Errorf("illegal package")
		return
	}
	// get heartbeart request from server
	if result.IsRequest {
		req := result.Result.(*remoting.Request)
		if req.Event {
			logger.Debugf("get rpc heartbeat request{%#v}", req)
			resp := remoting.NewResponse(req.ID, req.Version)
			resp.Status = hessian.Response_OK
			resp.Event = req.Event
			resp.SerialID = req.SerialID
			resp.Version = "2.0.2"
			reply(session, resp)
			return
		}
		logger.Errorf("illegal request but not heartbeart. {%#v}", req)
		return
	}
	h.timeoutTimes = 0
	p := result.Result.(*remoting.Response)
	// get heartbeart
	if p.Event {
		logger.Debugf("get rpc heartbeat response{%#v}", p)
		if p.Error != nil {
			logger.Errorf("rpc heartbeat response{error: %#v}", p.Error)
		}
		p.Handle()
		return
	}

	logger.Debugf("get rpc response{%#v}", p)

	h.conn.updateSession(session)

	p.Handle()
}

// OnCron check the session health periodic. if the session's sessionTimeout has reached, just close the session
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

	heartbeatCallBack := func(err error) {
		if err != nil {
			logger.Warnf("failed to send heartbeat, error{%v}", err)
			if h.timeoutTimes >= 3 {
				h.conn.removeSession(session)
				return
			}
			h.timeoutTimes++
			return
		}
		h.timeoutTimes = 0
	}

	if err := heartbeat(session, h.conn.pool.rpcClient.conf.heartbeatTimeout, heartbeatCallBack); err != nil {
		logger.Warnf("failed to send heartbeat, error{%v}", err)
	}
}

// //////////////////////////////////////////
// RpcServerHandler
// //////////////////////////////////////////

// nolint
type RpcServerHandler struct {
	maxSessionNum  int
	sessionTimeout time.Duration
	sessionMap     map[getty.Session]*rpcSession
	rwlock         sync.RWMutex
	server         *Server
	timeoutTimes   int
}

// nolint
func NewRpcServerHandler(maxSessionNum int, sessionTimeout time.Duration, serverP *Server) *RpcServerHandler {
	return &RpcServerHandler{
		maxSessionNum:  maxSessionNum,
		sessionTimeout: sessionTimeout,
		sessionMap:     make(map[getty.Session]*rpcSession),
		server:         serverP,
	}
}

// OnOpen call server session opened, add the session to getty server session list. also onOpen
// will check the max getty server session number
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

// OnError the getty server session has errored, so remove the session from the getty server session list
func (h *RpcServerHandler) OnError(session getty.Session, err error) {
	logger.Infof("session{%s} got error{%v}, will be closed.", session.Stat(), err)
	h.rwlock.Lock()
	delete(h.sessionMap, session)
	h.rwlock.Unlock()
}

// OnClose close the session, remove it from the getty server list
func (h *RpcServerHandler) OnClose(session getty.Session) {
	logger.Infof("session{%s} is closing......", session.Stat())
	h.rwlock.Lock()
	delete(h.sessionMap, session)
	h.rwlock.Unlock()
}

// OnMessage get request from getty client, update the session reqNum and reply response to client
func (h *RpcServerHandler) OnMessage(session getty.Session, pkg interface{}) {
	h.rwlock.Lock()
	if _, ok := h.sessionMap[session]; ok {
		h.sessionMap[session].reqNum++
	}
	h.rwlock.Unlock()

	decodeResult, ok := pkg.(remoting.DecodeResult)
	if !ok {
		logger.Errorf("illegal package{%#v}", pkg)
		return
	}
	if !decodeResult.IsRequest {
		res := decodeResult.Result.(*remoting.Response)
		if res.Event {
			logger.Debugf("get rpc heartbeat response{%#v}", res)
			if res.Error != nil {
				logger.Errorf("rpc heartbeat response{error: %#v}", res.Error)
			}
			res.Handle()
			return
		}
		logger.Errorf("illegal package but not heartbeart. {%#v}", pkg)
		return
	}
	req := decodeResult.Result.(*remoting.Request)

	resp := remoting.NewResponse(req.ID, req.Version)
	resp.Status = hessian.Response_OK
	resp.Event = req.Event
	resp.SerialID = req.SerialID
	resp.Version = "2.0.2"

	// heartbeat
	if req.Event {
		logger.Debugf("get rpc heartbeat request{%#v}", resp)
		reply(session, resp)
		return
	}

	defer func() {
		if e := recover(); e != nil {
			resp.Status = hessian.Response_SERVER_ERROR
			if err, ok := e.(error); ok {
				logger.Errorf("OnMessage panic: %+v", perrors.WithStack(err))
				resp.Error = perrors.WithStack(err)
			} else if err, ok := e.(string); ok {
				logger.Errorf("OnMessage panic: %+v", perrors.New(err))
				resp.Error = perrors.New(err)
			} else {
				logger.Errorf("OnMessage panic: %+v, this is impossible.", e)
				resp.Error = fmt.Errorf("OnMessage panic unknow exception. %+v", e)
			}

			if !req.TwoWay {
				return
			}
			reply(session, resp)
		}

	}()

	invoc, ok := req.Data.(*invocation.RPCInvocation)
	if !ok {
		panic("create invocation occur some exception for the type is not suitable one.")
	}
	attachments := invoc.Attachments()
	attachments[constant.LOCAL_ADDR] = session.LocalAddr()
	attachments[constant.REMOTE_ADDR] = session.RemoteAddr()

	result := h.server.requestHandler(invoc)
	if !req.TwoWay {
		return
	}
	resp.Result = result
	reply(session, resp)
}

// OnCron check the session health periodic. if the session's sessionTimeout has reached, just close the session
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

	heartbeatCallBack := func(err error) {
		if err != nil {
			logger.Warnf("failed to send heartbeat, error{%v}", err)
			if h.timeoutTimes >= 3 {
				h.rwlock.Lock()
				delete(h.sessionMap, session)
				h.rwlock.Unlock()
				session.Close()
				return
			}
			h.timeoutTimes++
			return
		}
		h.timeoutTimes = 0
	}

	if err := heartbeat(session, h.server.conf.heartbeatTimeout, heartbeatCallBack); err != nil {
		logger.Warnf("failed to send heartbeat, error{%v}", err)
	}
}

func reply(session getty.Session, resp *remoting.Response) {
	if err := session.WritePkg(resp, WritePkg_Timeout); err != nil {
		logger.Errorf("WritePkg error: %#v, %#v", perrors.WithStack(err), resp)
	}
}

func heartbeat(session getty.Session, timeout time.Duration, callBack func(err error)) error {
	req := remoting.NewRequest("2.0.2")
	req.TwoWay = true
	req.Event = true
	resp := remoting.NewPendingResponse(req.ID)
	remoting.AddPendingResponse(resp)
	err := session.WritePkg(req, 3*time.Second)

	go func() {
		var err1 error
		select {
		case <-getty.GetTimeWheel().After(timeout):
			err1 = errHeartbeatReadTimeout
		case <-resp.Done:
			err1 = resp.Err
		}
		callBack(err1)
	}()

	return perrors.WithStack(err)
}
