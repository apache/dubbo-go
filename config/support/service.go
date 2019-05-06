package support

import "github.com/dubbo/go-for-apache-dubbo/config"

var (
	conServices = map[string]config.RPCService{} // service name -> service
	proServices = map[string]config.RPCService{} // service name -> service
)

// SetConService is called by init() of implement of RPCService
func SetConService(service config.RPCService) {
	conServices[service.Service()] = service
}

// SetProService is called by init() of implement of RPCService
func SetProService(service config.RPCService) {
	proServices[service.Service()] = service
}

func GetConService(name string) config.RPCService {
	return conServices[name]
}

func GetProService(name string) config.RPCService {
	return proServices[name]
}
