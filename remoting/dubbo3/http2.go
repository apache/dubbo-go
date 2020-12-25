package dubbo3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
)

import (
	"github.com/gogo/protobuf/proto"
	perrors "github.com/pkg/errors"
	h2 "golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
	"google.golang.org/grpc"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
)

type H2Controller struct {
	conn      net.Conn
	rawFramer *h2.Framer
	isServer  bool

	streamID   int32
	streamMap  map[uint32]stream
	mdMap      map[string]grpc.MethodDesc
	strMap     map[string]grpc.StreamDesc
	url        *common.URL
	handler    remoting.ProtocolHeaderHandler
	pkgHandler remoting.PackageHandler
	service    common.RPCService

	sendChan chan interface{}
}

type sendChanDataPkg struct {
	data      []byte
	endStream bool
	streamID  uint32
}

func getMethodAndStreamDescMap(service common.RPCService) (map[string]grpc.MethodDesc, map[string]grpc.StreamDesc, error) {
	ds, ok := service.(Dubbo3GrpcService)
	if !ok {
		logger.Error("service is not Impl Dubbo3GrpcService")
		return nil, nil, perrors.New("service is not Impl Dubbo3GrpcService")
	}
	sdMap := make(map[string]grpc.MethodDesc, 8)
	strMap := make(map[string]grpc.StreamDesc, 8)
	for _, v := range ds.ServiceDesc().Methods {
		sdMap[v.MethodName] = v
	}
	for _, v := range ds.ServiceDesc().Streams {
		strMap[v.StreamName] = v
	}
	return sdMap, strMap, nil
}

func NewH2Controller(conn net.Conn, isServer bool, service common.RPCService, url *common.URL) (*H2Controller, error) {
	var mdMap map[string]grpc.MethodDesc
	var strMap map[string]grpc.StreamDesc
	var err error
	if isServer {
		mdMap, strMap, err = getMethodAndStreamDescMap(service)
		if err != nil {
			logger.Error("new H2 controller error:", err)
			return nil, err
		}
	}

	fm := h2.NewFramer(conn, conn)
	fm.ReadMetaHeaders = hpack.NewDecoder(4096, nil)
	var headerHandler remoting.ProtocolHeaderHandler
	var pkgHandler remoting.PackageHandler

	if url != nil {
		headerHandler, _ = remoting.GetProtocolHeaderHandler(url.Protocol)
		pkgHandler, _ = remoting.GetPackagerHandler(url.Protocol)
	}

	h2c := &H2Controller{
		rawFramer:  fm,
		conn:       conn,
		url:        url,
		isServer:   isServer,
		streamMap:  make(map[uint32]stream, 8),
		mdMap:      mdMap,
		strMap:     strMap,
		service:    service,
		handler:    headerHandler,
		pkgHandler: pkgHandler,
		sendChan:   make(chan interface{}, 16),
		streamID:   int32(-1),
	}
	return h2c, nil
}

func (h *H2Controller) H2ShakeHand() error {
	// todo 替换为真实settings
	settings := []h2.Setting{{
		ID:  0x5,
		Val: 16384,
	}}

	if h.isServer { // server
		// server端1：写入第一个空setting
		if err := h.rawFramer.WriteSettings(settings...); err != nil {
			logger.Error("server write setting frame error", err)
			return err
		}
		// server端2：读取magic
		// Check the validity of client preface.
		preface := make([]byte, len(h2.ClientPreface))
		if _, err := io.ReadFull(h.conn, preface); err != nil {
			logger.Error("server read preface err = ", err)
			return err
		}
		if !bytes.Equal(preface, []byte(h2.ClientPreface)) {
			logger.Error("server recv reface = ", string(preface), "not as expected")
			return perrors.Errorf("Preface Not Equal")
		}
		logger.Debug("server Preface check successful!")
		// server端3：读取第一个空setting
		frame, err := h.rawFramer.ReadFrame()
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			logger.Error("server read firest setting err = ", err)
			return err
		}
		if err != nil {
			logger.Error("server read setting err = ", err)
			return err
		}
		_, ok := frame.(*h2.SettingsFrame)
		if !ok {
			logger.Error("server read frame not setting frame type")
			return perrors.Errorf("server read frame not setting frame type")
		}
		if err := h.rawFramer.WriteSettingsAck(); err != nil {
			logger.Error("server write setting Ack() err = ", err)
			return err
		}
	} else { // client
		// client端1 写入magic
		if _, err := h.conn.Write([]byte(h2.ClientPreface)); err != nil {
			logger.Errorf("client write preface err = ", err)
			return err
		}

		// server端2：写入第一个空setting
		if err := h.rawFramer.WriteSettings(settings...); err != nil {
			logger.Errorf("client write setting frame err = ", err)
			return err
		}

		// client端3：读取第一个setting
		frame, err := h.rawFramer.ReadFrame()
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			logger.Error("client read setting err = ", err)
			return err
		}
		if err != nil {
			logger.Error("client read setting err = ", err)
			return err
		}
		_, ok := frame.(*h2.SettingsFrame)
		if !ok {
			logger.Error("client read frame not setting frame type")
			return perrors.Errorf("client read frame not setting frame type")
		}
	}
	go h.runSend()
	if h.isServer {
		go h.serverRunRecv()
	} else {
		go h.clientRunRecv()
	}
	return nil
}

// runSendUnaryRsp start a rsp loop,  called when server response unary rpc
func (h *H2Controller) runSendUnaryRsp(stream *serverStream) {
	sendChan := stream.getSend()
	for {
		sendMsg := <-sendChan
		sendData := sendMsg.buffer.Bytes()
		// header
		headerFields := make([]hpack.HeaderField, 0, 2) // at least :status, content-type will be there if none else.
		headerFields = append(headerFields, hpack.HeaderField{Name: ":status", Value: "200"})
		headerFields = append(headerFields, hpack.HeaderField{Name: "content-type", Value: "application/grpc"})

		var buf bytes.Buffer
		enc := hpack.NewEncoder(&buf)
		for _, f := range headerFields {
			if err := enc.WriteField(f); err != nil {
				logger.Error("error: enc.WriteField err = ", err)
			}
		}
		hflen := buf.Len()
		hfData := buf.Next(hflen)

		logger.Debug("server send stream id = ", stream.getID())

		h.sendChan <- h2.HeadersFrameParam{
			StreamID:      stream.getID(),
			EndHeaders:    true,
			BlockFragment: hfData,
			EndStream:     false,
		}
		h.sendChan <- sendChanDataPkg{
			streamID:  stream.getID(),
			endStream: false,
			data:      sendData,
		}

		// end stream
		buf.Reset()
		headerFields = make([]hpack.HeaderField, 0, 2) // at least :status, content-type will be there if none else.
		headerFields = append(headerFields, hpack.HeaderField{Name: "grpc-status", Value: "0"})
		headerFields = append(headerFields, hpack.HeaderField{Name: "grpc-message", Value: ""})
		for _, f := range headerFields {
			if err := enc.WriteField(f); err != nil {
				logger.Error("error: enc.WriteField err = ", err)
			}
		}
		hfData = buf.Next(buf.Len())
		// todo need thread safe
		h.sendChan <- h2.HeadersFrameParam{
			StreamID:      stream.getID(),
			EndHeaders:    true,
			BlockFragment: hfData,
			EndStream:     true,
		}
		break
	}
}

// runSendStreamRsp start a rsp loop,  called when server response stream rpc
func (h *H2Controller) runSendStreamRsp(stream *serverStream) {
	sendChan := stream.getSend()
	headerWrited := false
	for {
		sendMsg := <-sendChan
		if sendMsg.GetMsgType() == ServerStreamCloseMsgType {
			var buf bytes.Buffer
			enc := hpack.NewEncoder(&buf)
			headerFields := make([]hpack.HeaderField, 0, 2) // at least :status, content-type will be there if none else.
			headerFields = append(headerFields, hpack.HeaderField{Name: "grpc-status", Value: "0"})
			headerFields = append(headerFields, hpack.HeaderField{Name: "grpc-message", Value: ""})
			for _, f := range headerFields {
				if err := enc.WriteField(f); err != nil {
					logger.Error("error: enc.WriteField err = ", err)
				}
			}
			hfData := buf.Next(buf.Len())
			h.sendChan <- h2.HeadersFrameParam{
				StreamID:      stream.getID(),
				EndHeaders:    true,
				BlockFragment: hfData,
				EndStream:     true,
			}
			return
		}
		sendData := sendMsg.buffer.Bytes()
		if !headerWrited {
			// header
			headerFields := make([]hpack.HeaderField, 0, 2) // at least :status, content-type will be there if none else.
			headerFields = append(headerFields, hpack.HeaderField{Name: ":status", Value: "200"})
			headerFields = append(headerFields, hpack.HeaderField{Name: "content-type", Value: "application/grpc"})
			var buf bytes.Buffer
			enc := hpack.NewEncoder(&buf)
			for _, f := range headerFields {
				if err := enc.WriteField(f); err != nil {
					logger.Error("error: enc.WriteField err = ", err)
				}
			}
			hflen := buf.Len()
			hfData := buf.Next(hflen)

			logger.Debug("server send stream id = ", stream.getID())

			// todo need serilization
			h.sendChan <- h2.HeadersFrameParam{
				StreamID:      stream.getID(),
				EndHeaders:    true,
				BlockFragment: hfData,
				EndStream:     false,
			}
			headerWrited = true
		}

		h.sendChan <- sendChanDataPkg{
			streamID:  stream.getID(),
			endStream: false,
			data:      sendData,
		}
	}
}

func (h *H2Controller) clientRunRecv() {
	for {
		fm, err := h.rawFramer.ReadFrame()
		if err != nil {
			if err != io.EOF {
				logger.Info("read frame error = ", err)
			}
			continue
		}
		switch fm := fm.(type) {
		case *h2.MetaHeadersFrame:
			id := fm.StreamID
			logger.Debug("MetaHeader frame = ", fm.String(), "id = ", id)
			if fm.Flags.Has(h2.FlagDataEndStream) {
				// todo graceful close
				delete(h.streamMap, id)
				continue
			}
		case *h2.DataFrame:
			id := fm.StreamID
			logger.Debug("DataFrame = ", fm.String(), "id = ", id)
			data := make([]byte, len(fm.Data()))
			copy(data, fm.Data())
			h.streamMap[id].putRecv(data, DataMsgType)
		case *h2.RSTStreamFrame:
		case *h2.SettingsFrame:
		case *h2.PingFrame:
		case *h2.WindowUpdateFrame:
		case *h2.GoAwayFrame:
			// TODO: Handle GoAway from the client appropriately.
		default:
			//r := frame.(*h2.MetaHeadersFrame)
			//fmt.Println(r)
		}
	}
}

// serverRun start a loop, server start listening h2 metaheader
func (h *H2Controller) serverRunRecv() {
	var id uint32
	for {
		fm, err := h.rawFramer.ReadFrame()
		if err != nil {
			if err != io.EOF {
				logger.Info("read frame error = ", err)
			}
			continue
		}
		switch fm := fm.(type) {
		case *h2.MetaHeadersFrame:
			id = fm.StreamID
			logger.Debug("MetaHeader frame = ", fm.String(), "id = ", id)
			header := h.handler.ReadFromH2MetaHeader(fm)
			h.addServerStream(header)
		case *h2.DataFrame:
			logger.Debug("DataFrame = ", fm.String(), "id = ", id)
			data := make([]byte, len(fm.Data()))
			copy(data, fm.Data())
			h.streamMap[fm.StreamID].putRecv(data, DataMsgType)
		case *h2.RSTStreamFrame:
		case *h2.SettingsFrame:
		case *h2.PingFrame:
		case *h2.WindowUpdateFrame:
		case *h2.GoAwayFrame:
			// TODO: Handle GoAway from the client appropriately.
		default:
			//r := frame.(*h2.MetaHeadersFrame)
			//fmt.Println(r)
		}
	}
}
func (h *H2Controller) addServerStream(data remoting.ProtocolHeader) error {
	methodName := strings.Split(data.GetMethod(), "/")[2]
	md, okm := h.mdMap[methodName]
	streamd, oks := h.strMap[methodName]
	if !okm && !oks {
		logger.Errorf("method name %s not found in desc", methodName)
		return perrors.New(fmt.Sprintf("method name %s not found in desc", methodName))
	}
	var newstm *serverStream
	var err error
	if okm {
		newstm, err = newServerStream(data, md, h.url, h.service)
		if err != nil {
			return err
		}
		go h.runSendUnaryRsp(newstm)
	} else {
		newstm, err = newServerStream(data, streamd, h.url, h.service)
		if err != nil {
			return err
		}
		go h.runSendStreamRsp(newstm)
	}
	h.streamMap[data.GetStreamID()] = newstm
	return nil
}

func (h *H2Controller) runSend() {
	for {
		toSend := <-h.sendChan
		switch toSend := toSend.(type) {
		case h2.HeadersFrameParam:
			h.rawFramer.WriteHeaders(toSend)
		case sendChanDataPkg:
			h.rawFramer.WriteData(toSend.streamID, toSend.endStream, toSend.data)
		default:
		}
	}
}

func (h *H2Controller) StreamInvoke(ctx context.Context, method string) (grpc.ClientStream, error) {
	// metadata header
	handler, err := remoting.GetProtocolHeaderHandler(h.url.Protocol)
	if err != nil {
		return nil, err
	}
	h.url.SetParam(":path", method)
	headerFields := handler.WriteHeaderField(h.url, ctx)
	var buf bytes.Buffer
	enc := hpack.NewEncoder(&buf)
	for _, f := range headerFields {
		if err := enc.WriteField(f); err != nil {
			logger.Error("error: enc.WriteField err = ", err)
		} else {
			logger.Debug("encode field f = ", f.Name, " ", f.Value, " success")
		}
	}
	hflen := buf.Len()
	hfData := buf.Next(hflen)

	//todo rpcID change
	id := uint32(atomic.AddInt32(&h.streamID, 2))
	h.sendChan <- h2.HeadersFrameParam{
		StreamID:      id,
		EndHeaders:    true,
		BlockFragment: hfData,
		EndStream:     false,
	}
	clientStream := newClientStream(id, h.url)
	serilizer, err := remoting.GetDubbo3Serializer(defaultSerilization)
	if err != nil {
		logger.Error("get serilizer error = ", err)
		return nil, err
	}
	h.streamMap[clientStream.ID] = clientStream
	go h.runSendReq(clientStream)
	pkgHandler, err := remoting.GetPackagerHandler(h.url.Protocol)
	return newClientUserStream(clientStream, serilizer, pkgHandler), nil
}

// runSendReqAndRecv called when client stream invoke
func (h *H2Controller) runSendReq(stream *clientStream) error {
	sendChan := stream.getSend()
	for {
		sendMsg := <-sendChan
		sendData := sendMsg.buffer.Bytes()
		h.sendChan <- sendChanDataPkg{
			streamID:  stream.ID,
			endStream: false,
			data:      sendData,
		}
	}
}

func (h *H2Controller) UnaryInvoke(ctx context.Context, method string, addr string, data []byte, reply interface{}, url *common.URL) error {
	// 1. set client stream
	id := uint32(atomic.AddInt32(&h.streamID, 2))
	clientStream := newClientStream(id, url)
	h.streamMap[clientStream.ID] = clientStream

	// 2. send req messages

	// metadata header
	handler, err := remoting.GetProtocolHeaderHandler(url.Protocol)
	if err != nil {
		return err
	}
	// method name from grpc stub, set to :path field
	url.SetParam(":path", method)
	headerFields := handler.WriteHeaderField(url, ctx)
	var buf bytes.Buffer
	enc := hpack.NewEncoder(&buf)
	for _, f := range headerFields {
		if err := enc.WriteField(f); err != nil {
			logger.Error("error: enc.WriteField err = ", err)
		} else {
			logger.Debug("encode field f = ", f.Name, " ", f.Value, " success")
		}
	}
	hflen := buf.Len()
	hfData := buf.Next(hflen)

	// header send
	h.sendChan <- h2.HeadersFrameParam{
		StreamID:      id,
		EndHeaders:    true,
		BlockFragment: hfData,
		EndStream:     false,
	}

	// data send
	h.sendChan <- sendChanDataPkg{
		streamID:  id,
		endStream: true,
		data:      h.pkgHandler.Pkg2FrameData(data),
	}

	// recv rsp
	recvChan := clientStream.getRecv()
	recvData := <-recvChan
	if recvData.GetMsgType() != DataMsgType {
		return perrors.New("get data from req not data msg type")
	}

	if err := proto.Unmarshal(h.pkgHandler.Frame2PkgData(recvData.buffer.Bytes()), reply.(proto.Message)); err != nil {
		logger.Error("client unmarshal rsp ", err)
		return err
	}
	return nil
}
