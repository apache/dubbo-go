package main

import (
	"flag"

	"github.com/edison/go-telnet/client"

	"github.com/edison/go-telnet/json_register"
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
	flag.StringVar(&host, "host", "localhost", "target server host")
	flag.IntVar(&port, "port", 8080, "target server port")
	flag.StringVar(&protocolName, "protocol", "dubbo", "transfer protocol")
	flag.StringVar(&InterfaceID, "interface", "com", "target service registered interface")
	flag.StringVar(&version, "version", "", "target service version")
	flag.StringVar(&group, "group", "", "target service group")
	flag.StringVar(&method, "method", "", "target method")
	flag.StringVar(&sendObjFilePath, "sendObj", "", "json file path to define transfer struct")
	flag.StringVar(&recvObjFilePath, "recvObj", "", "json file path to define receive struct")
}

func main() {
	flag.Parse()
	reqPkg := json_register.RegisterStructFromFile(sendObjFilePath)
	recvPkg := json_register.RegisterStructFromFile(recvObjFilePath)

	t, err := client.NewTelnetClient(host, port, protocolName, InterfaceID, version, group, method, reqPkg)
	if err != nil {
		panic(err)
	}
	t.ProcessRequests(recvPkg)
}
