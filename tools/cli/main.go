package main

import (
	"flag"
	"log"
)

import (
	"github.com/apache/dubbo-go/tools/cli/client"
	"github.com/apache/dubbo-go/tools/cli/json_register"
)

var host string
var port int
var protocolName string
var InterfaceID string
var version string
var group string
var method string
var sendObjFilePath string
var recvObjFilePath string

func init() {
	flag.StringVar(&host, "h", "localhost", "target server host")
	flag.IntVar(&port, "p", 8080, "target server port")
	flag.StringVar(&protocolName, "proto", "dubbo", "transfer protocol")
	flag.StringVar(&InterfaceID, "i", "com", "target service registered interface")
	flag.StringVar(&version, "v", "", "target service version")
	flag.StringVar(&group, "g", "", "target service group")
	flag.StringVar(&method, "method", "", "target method")
	flag.StringVar(&sendObjFilePath, "sendObj", "", "json file path to define transfer struct")
	flag.StringVar(&recvObjFilePath, "recvObj", "", "json file path to define receive struct")
}

func checkParam() {
	if method == "" {
		log.Fatalln("-method value not fond")
	}
	if sendObjFilePath == "" {
		log.Fatalln("-sendObj value not found")
	}
	if recvObjFilePath == "" {
		log.Fatalln("-recObj value not found")
	}
}

func main() {
	flag.Parse()
	checkParam()
	reqPkg := json_register.RegisterStructFromFile(sendObjFilePath)
	recvPkg := json_register.RegisterStructFromFile(recvObjFilePath)

	t, err := client.NewTelnetClient(host, port, protocolName, InterfaceID, version, group, method, reqPkg)
	if err != nil {
		panic(err)
	}
	t.ProcessRequests(recvPkg)
	t.Destory()
}
