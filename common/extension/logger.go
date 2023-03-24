package extension

import (
	"github.com/pkg/errors"

	"github.com/dubbogo/gost/log/logger"

	"dubbo.apache.org/dubbo-go/v3/common"
)

var logs = make(map[string]func(config *common.URL) (logger.Logger, error))

func SetLogger(driver string, log func(config *common.URL) (logger.Logger, error)) {
	logs[driver] = log
}

func GetLogger(driver string, config *common.URL) (logger.Logger, error) {

	if logs[driver] != nil {
		return logs[driver](config)
	} else {
		return nil, errors.Errorf("logger for %s does not exist. "+
			"please make sure that you have imported the package "+
			"github.com/dubbogo/gost/log/logger/%s", driver, driver)
	}
}
