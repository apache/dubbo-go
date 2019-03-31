package server

import "github.com/AlexStocks/goext/net"

type ServerConfig struct {
	Protocol string `required:"true",default:"dubbo" yaml:"protocol" json:"protocol,omitempty"` // codec string, jsonrpc  etc
	IP       string `yaml:"ip" json:"ip,omitempty"`
	Port     int    `required:"true" yaml:"port" json:"port,omitempty"`
}

func (c *ServerConfig) Address() string {
	return gxnet.HostAddress(c.IP, c.Port)
}
