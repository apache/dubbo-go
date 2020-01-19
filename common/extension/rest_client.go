package extension

import (
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
)

var (
	restClients = make(map[string]func(restOptions *rest_interface.RestOptions) rest_interface.RestClient)
)

func SetRestClient(name string, fun func(restOptions *rest_interface.RestOptions) rest_interface.RestClient) {
	restClients[name] = fun
}

func GetRestClient(name string, restOptions *rest_interface.RestOptions) rest_interface.RestClient {
	if restClients[name] == nil {
		panic("restClient for " + name + " is not existing, make sure you have import the package.")
	}
	return restClients[name](restOptions)
}
