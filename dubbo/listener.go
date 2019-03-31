package dubbo

import (
	"time"
)

import (
	"github.com/AlexStocks/getty"
	log "github.com/AlexStocks/log4go"
	"github.com/dubbogo/hessian2"
	jerrors "github.com/juju/errors"
)

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
		log.Error("illegal packge")
		return
	}

	if p.Header.Type == hessian.Heartbeat {
		log.Debug("get rpc heartbeat response{header: %#v, body: %#v}", p.Header, p.Body)
		return
	}
	log.Debug("get rpc response{header: %#v, header: %#v}", p.Header, p.Body)

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
