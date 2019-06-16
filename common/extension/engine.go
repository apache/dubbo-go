package extension

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common"
)

var (
	engines = make(map[string]func(config *common.URL) (cluster.Router, error))
)

func SetEngine(name string, v func(config *common.URL) (cluster.Router, error)) {
	engines[name] = v
}

func GetEngine(name string, config *common.URL) (cluster.Router, error) {
	if engines[name] == nil {
		panic("registry for " + name + " is not existing, make sure you have import the package.")
	}
	return engines[name](config)

}
