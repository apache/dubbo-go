package support

import "github.com/dubbo/dubbo-go/config"

var (
	services = map[string]config.RPCService{} // service name -> service
)

// SetService is called by init() of implement of RPCService
func SetService(service config.RPCService) {
	services[service.Service()] = service
}

func GetService(name string) config.RPCService {
	return services[name]
}
