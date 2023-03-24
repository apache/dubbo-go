package logrus

import (
	"io"
	"os"
	"strings"

	"github.com/mattn/go-colorable"

	"github.com/sirupsen/logrus"

	"github.com/dubbogo/gost/log/logger"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	. "dubbo.apache.org/dubbo-go/v3/logger"
)

func init() {
	extension.SetLogger("logrus", instantiate)
}

type Logger struct {
	lg *logrus.Logger
}

func instantiate(config *common.URL) (log logger.Logger, err error) {
	var (
		level     string
		writer    []io.Writer
		lv        logrus.Level
		appender  []string
		formatter logrus.Formatter
		lg        *logrus.Logger
	)
	lg = logrus.New()
	level = config.GetParam(constant.LoggerLevelKey, constant.LoggerLevel)

	if lv, err = logrus.ParseLevel(level); err != nil {
		lg.SetLevel(logrus.InfoLevel)
	} else {
		lg.SetLevel(lv)
	}

	appender = strings.Split(config.GetParam(constant.LoggerAppenderKey, constant.LoggerAppender), ",")
	for _, apt := range appender {
		switch apt {
		case "console":
			writer = append(writer, os.Stdout)
		case "file":
			file := FileConfig(config)
			writer = append(writer, colorable.NewNonColorable(file))
		}
	}
	lg.SetOutput(io.MultiWriter(writer...))

	format := config.GetParam(constant.LoggerFormatKey, constant.LoggerFormat)
	switch strings.ToLower(format) {
	case "text":
		formatter = &logrus.TextFormatter{}
	case "json":
		formatter = &logrus.JSONFormatter{}
	default:
		formatter = &logrus.TextFormatter{}
	}
	lg.SetFormatter(formatter)
	return &Logger{lg: lg}, err
}

func (l *Logger) Debug(args ...interface{}) {
	l.lg.Debug(args)
}

func (l *Logger) Debugf(template string, args ...interface{}) {
	l.lg.Debugf(template, args)
}

func (l *Logger) Info(args ...interface{}) {
	l.lg.Info(args)
}

func (l *Logger) Infof(template string, args ...interface{}) {
	l.lg.Infof(template, args)
}

func (l *Logger) Warn(args ...interface{}) {
	l.lg.Warn(args)
}

func (l *Logger) Warnf(template string, args ...interface{}) {
	l.lg.Warnf(template, args)
}

func (l *Logger) Error(args ...interface{}) {
	l.lg.Error(args)
}

func (l *Logger) Errorf(template string, args ...interface{}) {
	l.lg.Errorf(template, args)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.lg.Fatal(args)
}

func (l *Logger) Fatalf(fmt string, args ...interface{}) {
	l.lg.Fatalf(fmt, args)
}
