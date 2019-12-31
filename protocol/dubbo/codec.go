package dubbo

import (
	"bufio"
	"encoding/binary"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common/logger"
)

type DubboCodec struct {
	reader     *bufio.Reader
	pkgType    PackageType
	bodyLen    int
	serializer Serializer
	headerRead bool
}

// enum part
const (
	PackageError              = PackageType(0x01)
	PackageRequest            = PackageType(0x02)
	PackageResponse           = PackageType(0x04)
	PackageHeartbeat          = PackageType(0x08)
	PackageRequest_TwoWay     = PackageType(0x10)
	PackageResponse_Exception = PackageType(0x20)
	PackageType_BitSize       = 0x2f
)

// call type
type CallType int32

const (
	CT_UNKNOWN CallType = 0
	CT_OneWay  CallType = 1
	CT_TwoWay  CallType = 2
)

type SequenceType int64

// PackageType ...
type PackageType int

func (c *DubboCodec) ReadHeader(header *DubboHeader) error {
	var err error
	logger.Infof("reader size: %v", c.reader.Size())
	if c.reader.Size() < HEADER_LENGTH {
		return hessian.ErrHeaderNotEnough
	}
	buf, err := c.reader.Peek(HEADER_LENGTH)
	if err != nil { // this is impossible
		return errors.WithStack(err)
	}
	_, err = c.reader.Discard(HEADER_LENGTH)
	if err != nil { // this is impossible
		return errors.WithStack(err)
	}

	//// read header
	if buf[0] != MAGIC_HIGH && buf[1] != MAGIC_LOW {
		return hessian.ErrIllegalPackage
	}

	// Header{serialization id(5 bit), event, two way, req/response}
	if header.SerialID = buf[2] & SERIAL_MASK; header.SerialID == Zero {
		return errors.Errorf("serialization ID:%v", header.SerialID)
	}

	flag := buf[2] & FLAG_EVENT
	if flag != Zero {
		header.Type |= PackageHeartbeat
	}
	flag = buf[2] & FLAG_REQUEST
	if flag != Zero {
		header.Type |= PackageRequest
		flag = buf[2] & FLAG_TWOWAY
		if flag != Zero {
			header.Type |= PackageRequest_TwoWay
		}
	} else {
		header.Type |= PackageResponse
		header.ResponseStatus = buf[3]
		if header.ResponseStatus != Response_OK {
			header.Type |= PackageResponse_Exception
		}
	}

	// Header{req id}
	header.ID = int64(binary.BigEndian.Uint64(buf[4:]))

	// Header{body len}
	header.BodyLen = int(binary.BigEndian.Uint32(buf[12:]))
	if header.BodyLen < 0 {
		return hessian.ErrIllegalPackage
	}

	c.pkgType = header.Type
	c.bodyLen = header.BodyLen

	if c.reader.Buffered() < c.bodyLen {
		return hessian.ErrBodyNotEnough
	}
	c.headerRead = true
	return errors.WithStack(err)
}

func (c *DubboCodec) EncodeHeader(p DubboPackage) []byte {
	header := p.Header
	bs := make([]byte, 0)
	switch header.Type {
	case PackageHeartbeat:
		if header.ResponseStatus == Zero {
			bs = append(bs, hessian.DubboRequestHeartbeatHeader[:]...)
		} else {
			bs = append(bs, hessian.DubboResponseHeartbeatHeader[:]...)
		}
	case PackageResponse:
		bs = append(bs, hessian.DubboResponseHeaderBytes[:]...)
		if header.ResponseStatus != 0 {
			bs[3] = header.ResponseStatus
		}
	case PackageRequest_TwoWay:
		bs = append(bs, hessian.DubboRequestHeaderBytesTwoWay[:]...)
	}
	bs[2] |= header.SerialID & hessian.SERIAL_MASK
	binary.BigEndian.PutUint64(bs[4:], uint64(header.ID))
	return bs
}

func (c *DubboCodec) Write(p DubboPackage) ([]byte, error) {
	// header
	if c.serializer == nil {
		return nil, errors.New("serializer should not be nil")
	}
	header := p.Header
	switch header.Type {
	case PackageHeartbeat:
		if header.ResponseStatus == Zero {
			return packRequest(p, c.serializer)
		}
		return packResponse(p, c.serializer)

	case PackageRequest, PackageRequest_TwoWay:
		return packRequest(p, c.serializer)

	case PackageResponse:
		return packResponse(p, c.serializer)

	default:
		return nil, errors.Errorf("Unrecognised message type: %v", header.Type)
	}
}

func (c *DubboCodec) Read(p *DubboPackage) error {
	if !c.headerRead {
		if err := c.ReadHeader(&p.Header); err != nil {
			return err
		}
	}
	body, err := c.reader.Peek(p.GetBodyLen())
	if err != nil {
		return err
	}
	if p.IsResponseWithException() {
		logger.Infof("response with exception: %+v", p.Header)
		decoder := hessian.NewDecoder(body)
		exception, err := decoder.Decode()
		if err != nil {
			return errors.WithStack(err)
		}
		rsp, ok := p.Body.(*ResponsePayload)
		if !ok {
			return errors.Errorf("java exception:%s", exception.(string))
		}
		rsp.Exception = errors.Errorf("java exception:%s", exception.(string))
		return nil
	} else if p.IsHeartBeat() {
		// heartbeat
		// heartbeat packag不去读取内容
		return nil
	}
	if c.serializer == nil {
		return errors.New("codec serializer is nil")
	}
	return c.serializer.Unmarshal(body, p)
}

func (c *DubboCodec) SetSerializer(serializer Serializer) {
	c.serializer = serializer
}

func packRequest(p DubboPackage, serializer Serializer) ([]byte, error) {
	var (
		byteArray []byte
		pkgLen    int
	)

	header := p.Header

	//////////////////////////////////////////
	// byteArray
	//////////////////////////////////////////
	// magic
	switch header.Type {
	case PackageHeartbeat:
		byteArray = append(byteArray, DubboRequestHeartbeatHeader[:]...)
	case PackageRequest_TwoWay:
		byteArray = append(byteArray, DubboRequestHeaderBytesTwoWay[:]...)
	default:
		byteArray = append(byteArray, DubboRequestHeaderBytes[:]...)
	}

	// serialization id, two way flag, event, request/response flag
	// SerialID is id of serialization approach in java dubbo
	byteArray[2] |= header.SerialID & SERIAL_MASK
	// request id
	binary.BigEndian.PutUint64(byteArray[4:], uint64(header.ID))

	//////////////////////////////////////////
	// body
	//////////////////////////////////////////
	if p.IsHeartBeat() {
		byteArray = append(byteArray, byte('N'))
		pkgLen = 1
	} else {
		body, err := serializer.Marshal(p)
		if err != nil {
			return nil, err
		}
		pkgLen = len(body)
		if pkgLen > int(DEFAULT_LEN) { // 8M
			return nil, errors.Errorf("Data length %d too large, max payload %d", pkgLen, DEFAULT_LEN)
		}
		byteArray = append(byteArray, body...)
	}
	binary.BigEndian.PutUint32(byteArray[12:], uint32(pkgLen))
	return byteArray, nil
}


func packResponse(p DubboPackage, serializer Serializer) ([]byte, error) {
	var (
		byteArray []byte
	)
	header := p.Header
	hb := p.IsHeartBeat()

	// magic
	if hb {
		byteArray = append(byteArray, DubboResponseHeartbeatHeader[:]...)
	} else {
		byteArray = append(byteArray, DubboResponseHeaderBytes[:]...)
	}
	// set serialID, identify serialization types, eg: fastjson->6, hessian2->2
	byteArray[2] |= header.SerialID & SERIAL_MASK
	// response status
	if header.ResponseStatus != 0 {
		byteArray[3] = header.ResponseStatus
	}

	// request id
	binary.BigEndian.PutUint64(byteArray[4:], uint64(header.ID))

	// body
	body, err := serializer.Marshal(p)
	if err != nil {
		return nil, err
	}

	//byteArray = encNull(byteArray) // if not, "java client" will throw exception  "unexpected end of file"
	pkgLen := len(body)
	if pkgLen > int(DEFAULT_LEN) { // 8M
		return nil, errors.Errorf("Data length %d too large, max payload %d", pkgLen, DEFAULT_LEN)
	}
	// byteArray{body length}
	binary.BigEndian.PutUint32(byteArray[12:], uint32(pkgLen))
	byteArray = append(byteArray, body...)
	return byteArray, nil
}

func NewDubboCodec(reader *bufio.Reader) *DubboCodec {
	return &DubboCodec{
		reader:  reader,
		pkgType: 0,
		bodyLen: 0,
		headerRead: false,
	}
}

type PendingResponse struct {
	seq       uint64
	err       error
	start     time.Time
	readStart time.Time
	callback  AsyncCallback
	response  *Response
	done      chan struct{}
}

func NewPendingResponse() *PendingResponse {
	return &PendingResponse{
		start:    time.Now(),
		response: &Response{},
		done:     make(chan struct{}),
	}
}

func (r PendingResponse) GetCallResponse() CallResponse {
	return CallResponse{
		Cause:     r.err,
		Start:     r.start,
		ReadStart: r.readStart,
		Reply:     r.response,
	}
}
