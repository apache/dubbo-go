/******************************************************
# DESC       : tcp/websocket connection
# MAINTAINER : Alex Stocks
# LICENCE    : Apache License 2.0
# EMAIL      : alexstocks@foxmail.com
# MOD        : 2016-08-17 11:21
# FILE       : conn.go
******************************************************/

package getty

import (
	"compress/flate"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	"github.com/golang/snappy"
	"github.com/gorilla/websocket"
	jerrors "github.com/juju/errors"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

var (
	launchTime = time.Now()

// ErrInvalidConnection = errors.New("connection has been closed.")
)

/////////////////////////////////////////
// getty connection
/////////////////////////////////////////

var (
	connID uint32
)

type gettyConn struct {
	id            uint32
	compress      CompressType
	padding1      uint8
	padding2      uint16
	readBytes     uint32        // read bytes
	writeBytes    uint32        // write bytes
	readPkgNum    uint32        // send pkg number
	writePkgNum   uint32        // recv pkg number
	active        int64         // last active, in milliseconds
	rTimeout      time.Duration // network current limiting
	wTimeout      time.Duration
	rLastDeadline time.Time // lastest network read time
	wLastDeadline time.Time // lastest network write time
	local         string    // local address
	peer          string    // peer address
	ss            Session
}

func (c *gettyConn) ID() uint32 {
	return c.id
}

func (c *gettyConn) LocalAddr() string {
	return c.local
}

func (c *gettyConn) RemoteAddr() string {
	return c.peer
}

func (c *gettyConn) incReadPkgNum() {
	atomic.AddUint32(&c.readPkgNum, 1)
}

func (c *gettyConn) incWritePkgNum() {
	atomic.AddUint32(&c.writePkgNum, 1)
}

func (c *gettyConn) UpdateActive() {
	atomic.StoreInt64(&(c.active), int64(time.Since(launchTime)))
}

func (c *gettyConn) GetActive() time.Time {
	return launchTime.Add(time.Duration(atomic.LoadInt64(&(c.active))))
}

func (c *gettyConn) Write(interface{}) (int, error) {
	return 0, nil
}

func (c *gettyConn) close(int) {}

func (c gettyConn) readTimeout() time.Duration {
	return c.rTimeout
}

func (c *gettyConn) setSession(ss Session) {
	c.ss = ss
}

// Pls do not set read deadline for websocket connection. AlexStocks 20180310
// gorilla/websocket/conn.go:NextReader will always fail when got a timeout error.
//
// Pls do not set read deadline when using compression. AlexStocks 20180314.
func (c *gettyConn) SetReadTimeout(rTimeout time.Duration) {
	if rTimeout < 1 {
		panic("@rTimeout < 1")
	}

	c.rTimeout = rTimeout
	if c.wTimeout == 0 {
		c.wTimeout = rTimeout
	}
}

func (c gettyConn) writeTimeout() time.Duration {
	return c.wTimeout
}

// Pls do not set write deadline for websocket connection. AlexStocks 20180310
// gorilla/websocket/conn.go:NextWriter will always fail when got a timeout error.
//
// Pls do not set write deadline when using compression. AlexStocks 20180314.
func (c *gettyConn) SetWriteTimeout(wTimeout time.Duration) {
	if wTimeout < 1 {
		panic("@wTimeout < 1")
	}

	c.wTimeout = wTimeout
	if c.rTimeout == 0 {
		c.rTimeout = wTimeout
	}
}

/////////////////////////////////////////
// getty tcp connection
/////////////////////////////////////////

type gettyTCPConn struct {
	gettyConn
	reader io.Reader
	writer io.Writer
	conn   net.Conn
}

// create gettyTCPConn
func newGettyTCPConn(conn net.Conn) *gettyTCPConn {
	if conn == nil {
		panic("newGettyTCPConn(conn):@conn is nil")
	}
	var localAddr, peerAddr string
	//  check conn.LocalAddr or conn.RemoetAddr is nil to defeat panic on 2016/09/27
	if conn.LocalAddr() != nil {
		localAddr = conn.LocalAddr().String()
	}
	if conn.RemoteAddr() != nil {
		peerAddr = conn.RemoteAddr().String()
	}

	return &gettyTCPConn{
		conn:   conn,
		reader: io.Reader(conn),
		writer: io.Writer(conn),
		gettyConn: gettyConn{
			id:       atomic.AddUint32(&connID, 1),
			rTimeout: netIOTimeout,
			wTimeout: netIOTimeout,
			local:    localAddr,
			peer:     peerAddr,
			compress: CompressNone,
		},
	}
}

// for zip compress
type writeFlusher struct {
	flusher *flate.Writer
	lock    sync.Mutex
}

func (t *writeFlusher) Write(p []byte) (int, error) {
	var (
		n   int
		err error
	)
	t.lock.Lock()
	defer t.lock.Unlock()
	n, err = t.flusher.Write(p)
	if err != nil {
		return n, jerrors.Trace(err)
	}
	if err := t.flusher.Flush(); err != nil {
		return 0, jerrors.Trace(err)
	}

	return n, nil
}

// set compress type(tcp: zip/snappy, websocket:zip)
func (t *gettyTCPConn) SetCompressType(c CompressType) {
	switch c {
	case CompressNone, CompressZip, CompressBestSpeed, CompressBestCompression, CompressHuffman:
		ioReader := io.Reader(t.conn)
		t.reader = flate.NewReader(ioReader)

		ioWriter := io.Writer(t.conn)
		w, err := flate.NewWriter(ioWriter, int(c))
		if err != nil {
			panic(fmt.Sprintf("flate.NewReader(flate.DefaultCompress) = err(%s)", err))
		}
		t.writer = &writeFlusher{flusher: w}

	case CompressSnappy:
		ioReader := io.Reader(t.conn)
		t.reader = snappy.NewReader(ioReader)
		ioWriter := io.Writer(t.conn)
		t.writer = snappy.NewBufferedWriter(ioWriter)

	default:
		panic(fmt.Sprintf("illegal comparess type %d", c))
	}
	t.compress = c
}

// tcp connection read
func (t *gettyTCPConn) read(p []byte) (int, error) {
	var (
		err         error
		currentTime time.Time
		length      int
	)

	if t.compress == CompressNone && t.rTimeout > 0 {
		// Optimization: update read deadline only if more than 25%
		// of the last read deadline exceeded.
		// See https://github.com/golang/go/issues/15133 for details.
		currentTime = wheel.Now()
		if currentTime.Sub(t.rLastDeadline) > (t.rTimeout >> 2) {
			if err = t.conn.SetReadDeadline(currentTime.Add(t.rTimeout)); err != nil {
				return 0, jerrors.Trace(err)
			}
			t.rLastDeadline = currentTime
		}
	}

	length, err = t.reader.Read(p)
	log.Debug("now:%s, length:%d, err:%s", currentTime, length, err)
	atomic.AddUint32(&t.readBytes, uint32(length))
	return length, jerrors.Trace(err)
	//return length, err
}

// tcp connection write
func (t *gettyTCPConn) Write(pkg interface{}) (int, error) {
	var (
		err         error
		currentTime time.Time
		ok          bool
		p           []byte
		length      int
	)

	if p, ok = pkg.([]byte); !ok {
		return 0, jerrors.Errorf("illegal @pkg{%#v} type", pkg)
	}
	if t.compress == CompressNone && t.wTimeout > 0 {
		// Optimization: update write deadline only if more than 25%
		// of the last write deadline exceeded.
		// See https://github.com/golang/go/issues/15133 for details.
		currentTime = wheel.Now()
		if currentTime.Sub(t.wLastDeadline) > (t.wTimeout >> 2) {
			if err = t.conn.SetWriteDeadline(currentTime.Add(t.wTimeout)); err != nil {
				return 0, jerrors.Trace(err)
			}
			t.wLastDeadline = currentTime
		}
	}

	if length, err = t.writer.Write(p); err == nil {
		atomic.AddUint32(&t.writeBytes, (uint32)(len(p)))
	}
	log.Debug("now:%s, length:%d, err:%s", currentTime, length, err)
	return length, jerrors.Trace(err)
	//return length, err
}

// close tcp connection
func (t *gettyTCPConn) close(waitSec int) {
	// if tcpConn, ok := t.conn.(*net.TCPConn); ok {
	// tcpConn.SetLinger(0)
	// }

	if t.conn != nil {
		if writer, ok := t.writer.(*snappy.Writer); ok {
			if err := writer.Close(); err != nil {
				log.Error("snappy.Writer.Close() = error{%s}", jerrors.ErrorStack(err))
			}
		}
		t.conn.(*net.TCPConn).SetLinger(waitSec)
		t.conn.Close()
		t.conn = nil
	}
}

/////////////////////////////////////////
// getty udp connection
/////////////////////////////////////////

type UDPContext struct {
	Pkg      interface{}
	PeerAddr *net.UDPAddr
}

func (c UDPContext) String() string {
	return fmt.Sprintf("{pkg:%#v, peer addr:%s}", c.Pkg, c.PeerAddr)
}

type gettyUDPConn struct {
	gettyConn
	compressType CompressType
	conn         *net.UDPConn // for server
}

func setUDPSocketOptions(conn *net.UDPConn) error {
	// Try setting the flags for both families and ignore the errors unless they
	// both error.
	err6 := ipv6.NewPacketConn(conn).SetControlMessage(ipv6.FlagDst|ipv6.FlagInterface, true)
	err4 := ipv4.NewPacketConn(conn).SetControlMessage(ipv4.FlagDst|ipv4.FlagInterface, true)
	if err6 != nil && err4 != nil {
		return jerrors.Trace(err4)
	}
	return nil
}

// create gettyUDPConn
func newGettyUDPConn(conn *net.UDPConn) *gettyUDPConn {
	if conn == nil {
		panic("newGettyUDPConn(conn):@conn is nil")
	}

	var localAddr, peerAddr string
	if conn.LocalAddr() != nil {
		localAddr = conn.LocalAddr().String()
	}

	if conn.RemoteAddr() != nil {
		// connected udp
		peerAddr = conn.RemoteAddr().String()
	}

	return &gettyUDPConn{
		conn: conn,
		gettyConn: gettyConn{
			id:       atomic.AddUint32(&connID, 1),
			rTimeout: netIOTimeout,
			wTimeout: netIOTimeout,
			local:    localAddr,
			peer:     peerAddr,
			compress: CompressNone,
		},
	}
}

func (u *gettyUDPConn) SetCompressType(c CompressType) {
	switch c {
	case CompressNone, CompressZip, CompressBestSpeed, CompressBestCompression, CompressHuffman, CompressSnappy:
		u.compressType = c

	default:
		panic(fmt.Sprintf("illegal comparess type %d", c))
	}
}

// udp connection read
func (u *gettyUDPConn) read(p []byte) (int, *net.UDPAddr, error) {
	var (
		err         error
		currentTime time.Time
		length      int
		addr        *net.UDPAddr
	)

	if u.rTimeout > 0 {
		// Optimization: update read deadline only if more than 25%
		// of the last read deadline exceeded.
		// See https://github.com/golang/go/issues/15133 for details.
		currentTime = wheel.Now()
		if currentTime.Sub(u.rLastDeadline) > (u.rTimeout >> 2) {
			if err = u.conn.SetReadDeadline(currentTime.Add(u.rTimeout)); err != nil {
				return 0, nil, jerrors.Trace(err)
			}
			u.rLastDeadline = currentTime
		}
	}

	length, addr, err = u.conn.ReadFromUDP(p) // connected udp also can get return @addr
	log.Debug("ReadFromUDP() = {length:%d, peerAddr:%s, error:%s}", length, addr, err)
	if err == nil {
		atomic.AddUint32(&u.readBytes, uint32(length))
	}

	//return length, addr, err
	return length, addr, jerrors.Trace(err)
}

// write udp packet, @ctx should be of type UDPContext
func (u *gettyUDPConn) Write(udpCtx interface{}) (int, error) {
	var (
		err         error
		currentTime time.Time
		length      int
		ok          bool
		ctx         UDPContext
		buf         []byte
		peerAddr    *net.UDPAddr
	)

	if ctx, ok = udpCtx.(UDPContext); !ok {
		return 0, jerrors.Errorf("illegal @udpCtx{%s} type, @udpCtx type:%T", udpCtx, udpCtx)
	}
	if buf, ok = ctx.Pkg.([]byte); !ok {
		return 0, jerrors.Errorf("illegal @udpCtx.Pkg{%#v} type", udpCtx)
	}
	if u.ss.EndPoint().EndPointType() == UDP_ENDPOINT {
		peerAddr = ctx.PeerAddr
		if peerAddr == nil {
			return 0, ErrNullPeerAddr
		}
	}

	if u.wTimeout > 0 {
		// Optimization: update write deadline only if more than 25%
		// of the last write deadline exceeded.
		// See https://github.com/golang/go/issues/15133 for details.
		currentTime = wheel.Now()
		if currentTime.Sub(u.wLastDeadline) > (u.wTimeout >> 2) {
			if err = u.conn.SetWriteDeadline(currentTime.Add(u.wTimeout)); err != nil {
				return 0, jerrors.Trace(err)
			}
			u.wLastDeadline = currentTime
		}
	}

	if length, _, err = u.conn.WriteMsgUDP(buf, nil, peerAddr); err == nil {
		atomic.AddUint32(&u.writeBytes, (uint32)(len(buf)))
	}
	log.Debug("WriteMsgUDP(peerAddr:%s) = {length:%d, error:%s}", peerAddr, length, err)

	return length, jerrors.Trace(err)
	//return length, err
}

// close udp connection
func (u *gettyUDPConn) close(_ int) {
	if u.conn != nil {
		u.conn.Close()
		u.conn = nil
	}
}

/////////////////////////////////////////
// getty websocket connection
/////////////////////////////////////////

type gettyWSConn struct {
	gettyConn
	conn *websocket.Conn
}

// create websocket connection
func newGettyWSConn(conn *websocket.Conn) *gettyWSConn {
	if conn == nil {
		panic("newGettyWSConn(conn):@conn is nil")
	}
	var localAddr, peerAddr string
	//  check conn.LocalAddr or conn.RemoetAddr is nil to defeat panic on 2016/09/27
	if conn.LocalAddr() != nil {
		localAddr = conn.LocalAddr().String()
	}
	if conn.RemoteAddr() != nil {
		peerAddr = conn.RemoteAddr().String()
	}

	gettyWSConn := &gettyWSConn{
		conn: conn,
		gettyConn: gettyConn{
			id:       atomic.AddUint32(&connID, 1),
			rTimeout: netIOTimeout,
			wTimeout: netIOTimeout,
			local:    localAddr,
			peer:     peerAddr,
			compress: CompressNone,
		},
	}
	conn.EnableWriteCompression(false)
	conn.SetPingHandler(gettyWSConn.handlePing)
	conn.SetPongHandler(gettyWSConn.handlePong)

	return gettyWSConn
}

// set compress type
func (w *gettyWSConn) SetCompressType(c CompressType) {
	switch c {
	case CompressNone, CompressZip, CompressBestSpeed, CompressBestCompression, CompressHuffman:
		w.conn.EnableWriteCompression(true)
		w.conn.SetCompressionLevel(int(c))

	default:
		panic(fmt.Sprintf("illegal comparess type %d", c))
	}
	w.compress = c
}

func (w *gettyWSConn) handlePing(message string) error {
	err := w.writePong([]byte(message))
	if err == websocket.ErrCloseSent {
		err = nil
	} else if e, ok := err.(net.Error); ok && e.Temporary() {
		err = nil
	}
	if err == nil {
		w.UpdateActive()
	}

	return jerrors.Trace(err)
}

func (w *gettyWSConn) handlePong(string) error {
	w.UpdateActive()
	return nil
}

// websocket connection read
func (w *gettyWSConn) read() ([]byte, error) {
	// Pls do not set read deadline when using ReadMessage. AlexStocks 20180310
	// gorilla/websocket/conn.go:NextReader will always fail when got a timeout error.
	_, b, e := w.conn.ReadMessage() // the first return value is message type.
	if e == nil {
		w.incReadPkgNum()
	} else {
		if websocket.IsUnexpectedCloseError(e, websocket.CloseGoingAway) {
			log.Warn("websocket unexpected close error: %v", e)
		}
	}

	return b, jerrors.Trace(e)
	//return b, e
}

func (w *gettyWSConn) updateWriteDeadline() error {
	var (
		err         error
		currentTime time.Time
	)

	if w.wTimeout > 0 {
		// Optimization: update write deadline only if more than 25%
		// of the last write deadline exceeded.
		// See https://github.com/golang/go/issues/15133 for details.
		currentTime = wheel.Now()
		if currentTime.Sub(w.wLastDeadline) > (w.wTimeout >> 2) {
			if err = w.conn.SetWriteDeadline(currentTime.Add(w.wTimeout)); err != nil {
				return jerrors.Trace(err)
			}
			w.wLastDeadline = currentTime
		}
	}

	return nil
}

// websocket connection write
func (w *gettyWSConn) Write(pkg interface{}) (int, error) {
	var (
		err error
		ok  bool
		p   []byte
	)

	if p, ok = pkg.([]byte); !ok {
		return 0, jerrors.Errorf("illegal @pkg{%#v} type", pkg)
	}

	w.updateWriteDeadline()
	if err = w.conn.WriteMessage(websocket.BinaryMessage, p); err == nil {
		atomic.AddUint32(&w.writeBytes, (uint32)(len(p)))
	}
	return len(p), jerrors.Trace(err)
	//return len(p), err
}

func (w *gettyWSConn) writePing() error {
	w.updateWriteDeadline()
	return jerrors.Trace(w.conn.WriteMessage(websocket.PingMessage, []byte{}))
}

func (w *gettyWSConn) writePong(message []byte) error {
	w.updateWriteDeadline()
	return jerrors.Trace(w.conn.WriteMessage(websocket.PongMessage, message))
}

// close websocket connection
func (w *gettyWSConn) close(waitSec int) {
	w.updateWriteDeadline()
	w.conn.WriteMessage(websocket.CloseMessage, []byte("bye-bye!!!"))
	conn := w.conn.UnderlyingConn()
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetLinger(waitSec)
	} else if wsConn, ok := conn.(*tls.Conn); ok {
		wsConn.CloseWrite()
	}
	w.conn.Close()
}
