package dubbo3

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
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

	streamMap map[uint32]*stream
	mdMap     map[string]grpc.MethodDesc
	url       *common.URL
	handler   remoting.ProtocolHeaderHandler
	service   common.RPCService
}

func getMethodDescMap(service common.RPCService) (map[string]grpc.MethodDesc, error) {
	ds, ok := service.(Dubbo3GrpcService)
	if !ok {
		logger.Error("service is not Impl Dubbo3GrpcService")
		return nil, perrors.New("service is not Impl Dubbo3GrpcService")
	}
	sdMap := make(map[string]grpc.MethodDesc, 8)
	for _, v := range ds.ServiceDesc().Methods {
		sdMap[v.MethodName] = v
	}
	return sdMap, nil
}

func NewH2Controller(conn net.Conn, isServer bool, service common.RPCService, url *common.URL) (*H2Controller, error) {
	mdMap := make(map[string]grpc.MethodDesc, 8)
	var err error
	if isServer {
		mdMap, err = getMethodDescMap(service)
		if err != nil {
			logger.Error("new H2 controller error:", err)
			return nil, err
		}
	}

	fm := h2.NewFramer(conn, conn)
	fm.ReadMetaHeaders = hpack.NewDecoder(4096, nil)
	var headerHandler remoting.ProtocolHeaderHandler

	if url != nil {
		headerHandler, _ = remoting.GetProtocolHeaderHandler(url.Protocol)
	}
	return &H2Controller{
		rawFramer: fm,
		conn:      conn,
		url:       url,
		isServer:  isServer,
		streamMap: make(map[uint32]*stream, 8),
		mdMap:     mdMap,
		service:   service,
		handler:   headerHandler,
	}, nil
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
	return nil

}

func (h *H2Controller) runSendRsp(stream *stream) {
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

		logger.Debug("stream id = ", stream.ID)

		if err := h.rawFramer.WriteHeaders(h2.HeadersFrameParam{
			StreamID:      stream.ID,
			EndHeaders:    true,
			BlockFragment: hfData,
			EndStream:     false,
		}); err != nil {
			logger.Error("error: write rsp header err = ", err)
		}
		if err := h.rawFramer.WriteData(stream.ID, false, sendData); err != nil {
			logger.Error("error: write rsp data err =", err)
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
		h.rawFramer.WriteHeaders(h2.HeadersFrameParam{
			StreamID:      stream.ID,
			EndHeaders:    true,
			BlockFragment: hfData,
			EndStream:     true,
		})
	}
}

func (h *H2Controller) run() {
	var id uint32
	for {
		fm, err := h.rawFramer.ReadFrame()
		if err != nil {
			if err != io.EOF {
				logger.Info("error = ", err)
			}
			break
		}
		switch fm := fm.(type) {
		case *h2.MetaHeadersFrame:
			id = fm.StreamID
			fmt.Println("MetaHeader frame = ", fm.String(), "id = ", id)
			h.handleMetaHeaderFrame(fm)
		case *h2.DataFrame:
			fmt.Println("DataFrame = ", fm.String(), "id = ", id)
			h.handleDataFrame(fm)
		case *h2.RSTStreamFrame:
			fmt.Println("RSTStreamFrame")
		case *h2.SettingsFrame:
			fmt.Println("SettingsFrame frame = ", fm.String())
		case *h2.PingFrame:
			fmt.Println("PingFrame")
		case *h2.WindowUpdateFrame:
			fmt.Println("WindowUpdateFrame")
		case *h2.GoAwayFrame:
			fmt.Println("GoAwayFrame")
			// TODO: Handle GoAway from the client appropriately.
		default:
			fmt.Println("default = %+v", fm)
			//r := frame.(*h2.MetaHeadersFrame)
			//fmt.Println(r)
		}
	}
}

func (h *H2Controller) handleMetaHeaderFrame(metaHeaderFrame *h2.MetaHeadersFrame) {
	header := h.handler.ReadFromH2MetaHeader(metaHeaderFrame)
	h.addStream(header)
}

func (h *H2Controller) handleDataFrame(fm *h2.DataFrame) {
	// 拿到stream，放进去
	h.streamMap[fm.StreamID].putRecv(fm.Data())
}

func (h *H2Controller) addStream(data remoting.ProtocolHeader) error {
	methodName := strings.Split(data.GetMethod(), "/")[2]
	// todo
	md, ok := h.mdMap[methodName]
	for k, v := range h.mdMap {
		fmt.Println(k, v)
	}
	if !ok {
		logger.Errorf("method name %s not found in method desc", methodName)
		return perrors.New(fmt.Sprintf("method name %s not found in method desc", methodName))
	}
	newStream, err := newStream(data, md, h.url, h.service)
	if err != nil {
		return err
	}
	h.streamMap[data.GetStreamID()] = newStream
	go h.runSendRsp(h.streamMap[data.GetStreamID()])
	return nil
}

func (h *H2Controller) UnaryInvoke(ctx context.Context, method string, addr string, data []byte, reply interface{}, url *common.URL) error {
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

	//	h.rawFramer.ReadMetaHeaders = hpack.NewDecoder(4096, nil)

	id := uint32(1)
	if err := h.rawFramer.WriteHeaders(h2.HeadersFrameParam{
		StreamID:      id,
		EndHeaders:    true,
		BlockFragment: hfData,
		EndStream:     false,
	}); err != nil {
		logger.Error("error: write rsp header err = ", err)
	}

	// todo 继续请求，发送数据，接受返回值

	//req := pb.HelloRequest{
	//	Name: "lauranceli",
	//	ID:   123,
	//	Subobj: &pb.HelloSubObj{
	//		SubName: "li",
	//	},
	//}

	header := make([]byte, 5)
	header[0] = 0
	binary.BigEndian.PutUint32(header[1:], uint32(len(data)))
	body := make([]byte, 5+len(data))
	copy(body[:5], header)
	copy(body[5:], data)
	if err := h.rawFramer.WriteData(id, true, body); err != nil {
		return err
	}

	for {
		fm, err := h.rawFramer.ReadFrame()
		if err != nil {
			logger.Error("unary invoke read frame error = ", err)
			break
		}
		switch fm := fm.(type) {
		case *h2.MetaHeadersFrame:
			id = fm.StreamID
			fmt.Println("MetaHeader frame = ", fm.String(), "id = ", id)
			parsedTripleHeaderData := parsedTripleHeaderData{}
			if fm.Flags.Has(h2.FlagDataEndStream) {
				return nil
			}
			parsedTripleHeaderData.FromMetaHeaderFrame(fm)

		case *h2.DataFrame:
			fmt.Println("DataFrame")
			frameData := fm.Data()
			fmt.Println(len(frameData))
			header := frameData[:5]
			length := binary.BigEndian.Uint32(header[1:])
			replyMessage := reply.(proto.Message)
			err := proto.Unmarshal(fm.Data()[5:5+length], replyMessage)
			if err != nil {
				logger.Error("clietn unmarshal rsp ", err)
				return err
			}
		case *h2.RSTStreamFrame:
			fmt.Println("RSTStreamFrame")
		case *h2.SettingsFrame:
			fmt.Println("SettingsFrame frame = ", fm.String())
		case *h2.PingFrame:
			fmt.Println("PingFrame")
		case *h2.WindowUpdateFrame:
			fmt.Println("WindowUpdateFrame")
		case *h2.GoAwayFrame:
			fmt.Println("GoAwayFrame")
			// TODO: Handle GoAway from the client appropriately.
		default:
			fmt.Println("default = %+v", fm)
			//r := frame.(*h2.MetaHeadersFrame)
			//fmt.Println(r)
		}
	}
	return nil
}
