package xds

import (
	"time"
)

import (
	xdsCommon "dubbo.apache.org/dubbo-go/v3/remoting/xds/common"
)

type Config struct {
	PodName         string
	Namespace       string
	IstioAddr       xdsCommon.HostAddr
	DebugPort       string
	LocalIP         string
	LocalDebugMode  bool
	SniffingTimeout time.Duration
}
