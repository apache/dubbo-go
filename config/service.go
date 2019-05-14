package config

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
)

var (
	conServices = map[string]common.RPCService{} // service name -> service
	proServices = map[string]common.RPCService{} // service name -> service
)

// SetConService is called by init() of implement of RPCService
func SetConService(service common.RPCService) {
	conServices[service.Service()] = service
}

// SetProService is called by init() of implement of RPCService
func SetProService(service common.RPCService) {
	proServices[service.Service()] = service
}

func GetConService(name string) common.RPCService {
	return conServices[name]
}

func GetProService(name string) common.RPCService {
	return proServices[name]
}
