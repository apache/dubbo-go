package logger

import (
	"io"
	"os"
)

import (
	getty "github.com/apache/dubbo-getty"
)

import (
	"github.com/sirupsen/logrus"
)

var baseLogrusLogger *logrus.Logger

func InitLogrusLog(logConfFile string) error {
	baseLogrusLogger = &logrus.Logger{
		Out: os.Stderr,
		Formatter: &logrus.TextFormatter{
			DisableColors: false,
			FullTimestamp: true,
		},
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.DebugLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}
	logger = &DubboLogger{Logger: baseLogrusLogger, dynamicLevel: logrus.DebugLevel.String(), loggerName: "logrus"}
	// set getty log
	getty.SetLogger(logger)
	return nil
}

func SetLogrusFormatter(formatter logrus.Formatter) {
	baseLogrusLogger.SetFormatter(formatter)
}

func SetLogrusLevel(level string) {
	l := new(logrus.Level)
	if err := l.UnmarshalText([]byte(level)); err == nil {
		baseLogrusLogger.SetLevel(*l)
	}
}

func SetLogrusOutput(opt io.Writer) {
	baseLogrusLogger.SetOutput(opt)
}

func SetLogrusReportCaller(reportCaller bool) {
	baseLogrusLogger.SetReportCaller(reportCaller)
}

func SetLogrusNoLock() {
	baseLogrusLogger.SetNoLock()
}
