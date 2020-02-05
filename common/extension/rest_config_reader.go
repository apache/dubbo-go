package extension

import (
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
)

var (
	restConfigReaders = make(map[string]func() rest_interface.RestConfigReader)
)

func SetRestConfigReader(name string, fun func() rest_interface.RestConfigReader) {
	restConfigReaders[name] = fun
}

func GetSingletonRestConfigReader(name string) rest_interface.RestConfigReader {
	if name == "" {
		name = "default"
	}
	if restConfigReaders[name] == nil {
		panic("restConfigReaders for " + name + " is not existing, make sure you have import the package.")
	}
	return restConfigReaders[name]()

}
