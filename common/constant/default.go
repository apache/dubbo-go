package constant

const (
	DEFAULT_WEIGHT = 100     //
	DEFAULT_WARMUP = 10 * 60 // in java here is 10*60*1000 because of System.currentTimeMillis() is measured in milliseconds & in go time.Unix() is second
)

const (
	DEFAULT_LOADBALANCE = "random"
	DEFAULT_RETRIES     = 2
	DEFAULT_PROTOCOL    = "dubbo"
	DEFAULT_VERSION     = ""
	DEFAULT_REG_TIMEOUT = "10s"
	DEFAULT_CLUSTER     = "failover"
)

const (
	ECHO = "$echo"
)
