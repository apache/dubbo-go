package metrics

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"fmt"
)

func getInvokerKey(url *common.URL) string {
	return url.Path
}

func getInstanceKey(url *common.URL) string {
	return fmt.Sprintf("%s:%s", url.Ip, url.Port)
}