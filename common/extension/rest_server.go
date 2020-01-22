package extension

import (
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
)

var (
	restServers = make(map[string]func() rest_interface.RestServer)
)

func SetRestServerFunc(name string, fun func() rest_interface.RestServer) {
	restServers[name] = fun
}

func GetNewRestServer(name string) rest_interface.RestServer {
	if restServers[name] == nil {
		panic("restServer for " + name + " is not existing, make sure you have import the package.")
	}
	return restServers[name]()
}
