/******************************************************
# DESC    : const properties
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2018-03-17 16:54
# FILE    : const.go
******************************************************/

package getty

import (
	"strconv"
)

type EndPointType int32

const (
	UDP_ENDPOINT EndPointType = 0
	UDP_CLIENT   EndPointType = 1
	TCP_CLIENT   EndPointType = 2
	WS_CLIENT    EndPointType = 3
	WSS_CLIENT   EndPointType = 4
	TCP_SERVER   EndPointType = 7
	WS_SERVER    EndPointType = 8
	WSS_SERVER   EndPointType = 9
)

var EndPointType_name = map[int32]string{
	0: "UDP_ENDPOINT",
	1: "UDP_CLIENT",
	2: "TCP_CLIENT",
	3: "WS_CLIENT",
	4: "WSS_CLIENT",
	7: "TCP_SERVER",
	8: "WS_SERVER",
	9: "WSS_SERVER",
}

var EndPointType_value = map[string]int32{
	"UDP_ENDPOINT": 0,
	"UDP_CLIENT":   1,
	"TCP_CLIENT":   2,
	"WS_CLIENT":    3,
	"WSS_CLIENT":   4,
	"TCP_SERVER":   7,
	"WS_SERVER":    8,
	"WSS_SERVER":   9,
}

func (x EndPointType) String() string {
	s, ok := EndPointType_name[int32(x)]
	if ok {
		return s
	}

	return strconv.Itoa(int(x))
}
