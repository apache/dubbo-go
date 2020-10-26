package common

import gxnet "github.com/dubbogo/gost/net"

var localIp string

func GetLocalIp() string {
	if len(localIp) != 0 {
		return localIp
	}
	localIp, _ = gxnet.GetLocalIP()
	return localIp
}
