package logger

import (
	"gopkg.in/natefinch/lumberjack.v2"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func FileConfig(config *common.URL) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   config.GetParam(constant.LoggerFileNameKey, "dubbo.log"),
		MaxSize:    config.GetParamByIntValue(constant.LoggerFileNaxSizeKey, 1),
		MaxBackups: config.GetParamByIntValue(constant.LoggerFileMaxBackupsKey, 1),
		MaxAge:     config.GetParamByIntValue(constant.LoggerFileMaxAgeKey, 3),
		LocalTime:  config.GetParamBool(constant.LoggerFileLocalTimeKey, true),
		Compress:   config.GetParamBool(constant.LoggerFileCompressKey, true),
	}
}
