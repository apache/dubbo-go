package client

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/edison/go-telnet/common"

	"github.com/edison/go-telnet/protocol"

	"go.uber.org/atomic"

	_ "github.com/edison/go-telnet/protocol/dubbo"
)

const defaultBufferSize = 4096

// TelnetClient represents a TCP client which is responsible for writing input data and printing response.
type TelnetClient struct {
	responseTimeout time.Duration
	protocolName    string
	requestList     []*protocol.Request
	conn            *net.TCPConn
	proto           protocol.Protocol

	sequence         atomic.Uint64
	pendingResponses *sync.Map
}

// PendingResponse is a pending response.
type PendingResponse struct {
	Seq       uint64
	err       error
	start     time.Time
	readStart time.Time
	Response  *Response
	done      chan struct{}
}

// Response is protocol protocol response.
type Response struct {
	Reply interface{}
	atta  map[string]string
}

// NewResponse creates a new Response.
func NewResponse(reply interface{}, atta map[string]string) *Response {
	return &Response{
		Reply: reply,
		atta:  atta,
	}
}

// NewTelnetClient method creates new instance of TCP client.
func NewTelnetClient(host string, port int, protocolName, interfaceID, version, group, method string, reqPkg interface{}) (*TelnetClient, error) {
	tcpAddr := createTCPAddr(host, port)
	resolved := resolveTCPAddr(tcpAddr)
	conn, err := net.DialTCP("tcp", nil, resolved)
	if err != nil {
		return nil, err
	}
	proto := common.GetProtocol(protocolName)

	return &TelnetClient{
		conn:             conn,
		responseTimeout:  100000000, //default timeout
		protocolName:     protocolName,
		pendingResponses: &sync.Map{},
		proto:            proto,
		requestList: []*protocol.Request{
			{
				InterfaceID: interfaceID,
				Version:     version,
				Method:      method,
				Group:       group,
				Params:      []interface{}{reqPkg},
			},
		},
	}, nil
}

func createTCPAddr(host string, port int) string {
	var buffer bytes.Buffer
	buffer.WriteString(host)
	buffer.WriteByte(':')
	buffer.WriteString(fmt.Sprintf("%d", port))
	return buffer.String()
}

func resolveTCPAddr(addr string) *net.TCPAddr {
	resolved, error := net.ResolveTCPAddr("tcp", addr)
	if nil != error {
		log.Fatalf("Error occured while resolving TCP address \"%v\": %v\n", addr, error)
	}

	return resolved
}

// 依次发起所有请求
func (t *TelnetClient) ProcessRequests(userPkg interface{}) {
	for i, _ := range t.requestList {
		t.processSingleRequest(t.requestList[i], userPkg)
	}
}

func (t *TelnetClient) addPendingResponse(model interface{}) uint64 {
	seqId := t.sequence.Load()
	t.pendingResponses.Store(seqId, model)
	fmt.Println("t.sequenceID = ", t.sequence)
	t.sequence.Inc()
	return seqId
}

func (t *TelnetClient) removePendingResponse(seq uint64) {
	if t.pendingResponses == nil {
		return
	}
	if _, ok := t.pendingResponses.Load(seq); ok {
		t.pendingResponses.Delete(seq)
	}
	return
}

// ProcessData method processes data: reads from input and writes to output.
func (t *TelnetClient) processSingleRequest(req *protocol.Request, userPkg interface{}) {
	// 协议打包过程
	req.ID = t.sequence.Load()
	inputData, err := t.proto.Write(req)
	if err != nil {
		log.Println("error: handler.Writer err = ", err)
	}

	// 2. 如果有需要，先初始化协议异步回包
	seqId := t.addPendingResponse(userPkg)
	defer t.removePendingResponse(seqId)

	// （三）开启链接过程

	requestDataChannel := make(chan []byte)
	doneChannel := make(chan bool)
	responseDataChannel := make(chan []byte)

	// （四）数据传输监听过程
	go t.readInputData(string(inputData), requestDataChannel, doneChannel)
	go t.readServerData(t.conn, responseDataChannel)

	var afterEOFResponseTicker = new(time.Ticker)
	var afterEOFMode bool
	var somethingRead bool

	for {
		select {
		case request := <-requestDataChannel:
			if _, error := t.conn.Write(request); nil != error {
				log.Fatalf("Error occured while writing to TCP socket: %v\n", error)
			}
		case <-doneChannel:
			afterEOFMode = true
			afterEOFResponseTicker = time.NewTicker(t.responseTimeout)
		case response := <-responseDataChannel:
			// （五）协议解包过程
			rspPkg, _, err := t.proto.Read(response, t.pendingResponses)
			if err != nil {
				fmt.Println("error with handler.Read ", err)
				break
			}
			fmt.Println(rspPkg)
			val := reflect.ValueOf(rspPkg).Elem()
			typ := reflect.TypeOf(rspPkg).Elem()
			nums := val.NumField()
			for i := 0; i < nums; i++ {
				if typ.Field(i).Type.Kind() == reflect.Ptr {
					fmt.Printf("subINfo = %+v", val.Field(i).Interface())
				}
			}
			if afterEOFMode {
				afterEOFResponseTicker.Stop()
				afterEOFResponseTicker = time.NewTicker(t.responseTimeout)
			}
		case <-afterEOFResponseTicker.C:
			if !somethingRead {
				log.Println("Nothing read. Maybe connection timeout.")
			}
			return
		}
	}
}

func (t *TelnetClient) readInputData(inputData string, toSent chan<- []byte, doneChannel chan<- bool) {
	toSent <- []byte(inputData)
	//t.assertEOF(error)
	doneChannel <- true
}

func (t *TelnetClient) readServerData(connection *net.TCPConn, received chan<- []byte) {
	buffer := make([]byte, defaultBufferSize)
	var error error
	var n int

	for nil == error {
		n, error = connection.Read(buffer)
		received <- buffer[:n]
	}

	t.assertEOF(error)
}

func (t *TelnetClient) assertEOF(error error) {
	if "EOF" != error.Error() {
		log.Fatalf("Error occured while operating on TCP socket: %v\n", error)
	}
}

func (t *TelnetClient) Destory() {
	t.conn.Close()
}
