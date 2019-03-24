/******************************************************
# DESC       : getty interface
# MAINTAINER : Alex Stocks
# LICENCE    : Apache License 2.0
# EMAIL      : alexstocks@foxmail.com
# MOD        : 2016-08-17 11:20
# FILE       : getty.go
******************************************************/

package getty

import (
	"compress/flate"
	"errors"
	"net"
	"time"
)

// NewSessionCallback will be invoked when server accepts a new client connection or client connects to server successfully.
// If there are too many client connections or u do not want to connect a server again, u can return non-nil error. And
// then getty will close the new session.
type NewSessionCallback func(Session) error

// Reader is used to unmarshal a complete pkg from buffer
type Reader interface {
	// Parse tcp/udp/websocket pkg from buffer and if possible return a complete pkg
	// If length of buf is not long enough, u should return {nil,0, nil}
	// The second return value is the length of the pkg.
	Read(Session, []byte) (interface{}, int, error)
}

// Writer is used to marshal pkg and write to session
type Writer interface {
	// if @Session is udpGettySession, the second parameter is UDPContext.
	Write(Session, interface{}) error
}

// tcp package handler interface
type ReadWriter interface {
	Reader
	Writer
}

// EventListener is used to process pkg that received from remote session
type EventListener interface {
	// invoked when session opened
	// If the return error is not nil, @Session will be closed.
	OnOpen(Session) error

	// invoked when session closed.
	OnClose(Session)

	// invoked when got error.
	OnError(Session, error)

	// invoked periodically, its period can be set by (Session)SetCronPeriod
	OnCron(Session)

	// invoked when receive packge. Pls attention that do not handle long time logic processing in this func.
	// You'd better set the package's maximum length. If the message's length is greater than it, u should
	// should return err in Reader{Read} and getty will close this connection soon.
	//
	// If this is a udp event listener, the second parameter type is UDPContext.
	OnMessage(Session, interface{})
}

/////////////////////////////////////////
// compress
/////////////////////////////////////////

type CompressType int

const (
	CompressNone            CompressType = flate.NoCompression      // 0
	CompressZip                          = flate.DefaultCompression // -1
	CompressBestSpeed                    = flate.BestSpeed          // 1
	CompressBestCompression              = flate.BestCompression    // 9
	CompressHuffman                      = flate.HuffmanOnly        // -2
	CompressSnappy                       = 10
)

/////////////////////////////////////////
// connection interfacke
/////////////////////////////////////////

type Connection interface {
	ID() uint32
	SetCompressType(CompressType)
	LocalAddr() string
	RemoteAddr() string
	incReadPkgNum()
	incWritePkgNum()
	// update session's active time
	UpdateActive()
	// get session's active time
	GetActive() time.Time
	readTimeout() time.Duration
	// SetReadTimeout sets deadline for the future read calls.
	SetReadTimeout(time.Duration)
	writeTimeout() time.Duration
	// SetWriteTimeout sets deadline for the future read calls.
	SetWriteTimeout(time.Duration)
	Write(interface{}) (int, error)
	// don't distinguish between tcp connection and websocket connection. Because
	// gorilla/websocket/conn.go:(Conn)Close also invoke net.Conn.Close
	close(int)
	// set related session
	setSession(Session)
}

/////////////////////////////////////////
// Session interfacke
/////////////////////////////////////////

var (
	ErrSessionClosed  = errors.New("session Already Closed")
	ErrSessionBlocked = errors.New("session Full Blocked")
	ErrNullPeerAddr   = errors.New("peer address is nil")
)

type Session interface {
	Connection
	Reset()
	Conn() net.Conn
	Stat() string
	IsClosed() bool
	// get endpoint type
	EndPoint() EndPoint

	SetMaxMsgLen(int)
	SetName(string)
	SetEventListener(EventListener)
	SetPkgHandler(ReadWriter)
	SetReader(Reader)
	SetWriter(Writer)
	SetCronPeriod(int)
	SetRQLen(int)
	SetWQLen(int)
	SetWaitTime(time.Duration)

	GetAttribute(interface{}) interface{}
	SetAttribute(interface{}, interface{})
	RemoveAttribute(interface{})

	// the Writer will invoke this function. Pls attention that if timeout is less than 0, WritePkg will send @pkg asap.
	// for udp session, the first parameter should be UDPContext.
	WritePkg(pkg interface{}, timeout time.Duration) error
	WriteBytes([]byte) error
	WriteBytesArray(...[]byte) error
	Close()
}

/////////////////////////////////////////
// EndPoint interfacke
/////////////////////////////////////////

type EndPoint interface {
	// get endpoint type
	EndPointType() EndPointType
	// run event loop and serves client request.
	RunEventLoop(newSession NewSessionCallback)
	// check the endpoint has been closed
	IsClosed() bool
	// close the endpoint and free its resource
	Close()
}

type Client interface {
	EndPoint
}

type Server interface {
	EndPoint
	// get the network listener
	Listener() net.Listener
}
