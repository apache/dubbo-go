package client

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

import (
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/tools/cli/common"
	"github.com/apache/dubbo-go/tools/cli/protocol"
	_ "github.com/apache/dubbo-go/tools/cli/protocol/dubbo"
)

const defaultBufferSize = 4096

// TelnetClient，维持一个服务的链接。
type TelnetClient struct {
	responseTimeout time.Duration
	protocolName    string
	requestList     []*protocol.Request
	conn            *net.TCPConn
	proto           protocol.Protocol

	sequence         atomic.Uint64
	pendingResponses *sync.Map
	waitNum          atomic.Uint64
}

// NewTelnetClient 创建一个新的tcp链接，并初始化默认请求
func NewTelnetClient(host string, port int, protocolName, interfaceID, version, group, method string, reqPkg interface{}) (*TelnetClient, error) {
	tcpAddr := createTCPAddr(host, port)
	resolved := resolveTCPAddr(tcpAddr)
	conn, err := net.DialTCP("tcp", nil, resolved)
	if err != nil {
		return nil, err
	}
	log.Printf("connected to %s:%d!\n", host, port)
	log.Printf("try calling interface:%s.%s\n", interfaceID, method)
	log.Printf("with protocol:%s\n\n", protocolName)
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

// ProcessRequests 依次发起所有请求
func (t *TelnetClient) ProcessRequests(userPkg interface{}) {
	for i, _ := range t.requestList {
		t.processSingleRequest(t.requestList[i], userPkg)
	}
}

// addPendingResponse 加入等待队列
func (t *TelnetClient) addPendingResponse(model interface{}) uint64 {
	seqId := t.sequence.Load()
	t.pendingResponses.Store(seqId, model)
	t.waitNum.Inc()
	t.sequence.Inc()
	return seqId
}

// removePendingResponse 从等待队列中删除
func (t *TelnetClient) removePendingResponse(seq uint64) {
	if t.pendingResponses == nil {
		return
	}
	if _, ok := t.pendingResponses.Load(seq); ok {
		t.pendingResponses.Delete(seq)
		t.waitNum.Dec()
	}
	return
}

// processSingleRequest 执行单个request
func (t *TelnetClient) processSingleRequest(req *protocol.Request, userPkg interface{}) {
	// 协议打包过程
	req.ID = t.sequence.Load()
	inputData, err := t.proto.Write(req)
	if err != nil {
		log.Fatalln("error: handler.Writer err = ", err)
	}
	startTime := time.Now()

	// 如果有需要，先初始化协议异步回包
	seqId := t.addPendingResponse(userPkg)
	defer t.removePendingResponse(seqId)

	requestDataChannel := make(chan []byte)
	doneChannel := make(chan bool)
	responseDataChannel := make(chan []byte)

	// 数据传输监听过程
	go t.readInputData(string(inputData), requestDataChannel, doneChannel)
	go t.readServerData(t.conn, responseDataChannel)

	for {
		select {
		case request := <-requestDataChannel:
			if _, err := t.conn.Write(request); nil != err {
				log.Fatalf("Error occured while writing to TCP socket: %v\n", err)
			}
		case response := <-responseDataChannel:
			rspPkg, _, err := t.proto.Read(response, t.pendingResponses)
			if err != nil {
				log.Fatalln("Error with protocol Read(): ", err)
			}
			t.removePendingResponse(seqId)
			log.Printf("After %dms , Got Rsp:", time.Now().Sub(startTime).Milliseconds())
			common.PrintInterface(rspPkg)
			if t.waitNum.Sub(0) == 0 {
				return
			}
		}
	}
}

func (t *TelnetClient) readInputData(inputData string, toSent chan<- []byte, doneChannel chan<- bool) {
	toSent <- []byte(inputData)
	doneChannel <- true
}

func (t *TelnetClient) readServerData(connection *net.TCPConn, received chan<- []byte) {
	buffer := make([]byte, defaultBufferSize)
	var err error
	var n int
	for nil == err {
		n, err = connection.Read(buffer)
		received <- buffer[:n]
	}

	t.assertEOF(err)
}

func (t *TelnetClient) assertEOF(error error) {
	if "EOF" != error.Error() {
		log.Fatalf("Error occured while operating on TCP socket: %v\n", error)
	}
}

func (t *TelnetClient) Destory() {
	t.conn.Close()
}
