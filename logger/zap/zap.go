package zap

import (
	"os"
	"strings"

	"github.com/mattn/go-colorable"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/dubbogo/gost/log/logger"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	. "dubbo.apache.org/dubbo-go/v3/logger"
)

func init() {
	extension.SetLogger("zap", instantiate)
}

func instantiate(config *common.URL) (log logger.Logger, err error) {
	var (
		level    string
		lv       zapcore.Level
		sync     []zapcore.WriteSyncer
		encoder  zapcore.Encoder
		appender []string
	)

	level = config.GetParam(constant.LoggerLevelKey, constant.LoggerLevel)
	if lv, err = zapcore.ParseLevel(level); err != nil {
		return nil, err
	}

	appender = strings.Split(config.GetParam(constant.LoggerAppenderKey, constant.LoggerAppender), ",")
	for _, apt := range appender {
		switch apt {
		case "console":
			sync = append(sync, zapcore.AddSync(os.Stdout))
		case "file":
			file := FileConfig(config)
			sync = append(sync, zapcore.AddSync(colorable.NewNonColorable(file)))
		}
	}

	format := config.GetParam(constant.LoggerFormatKey, constant.LoggerFormat)
	switch strings.ToLower(format) {
	case "text":
		encoder = zapcore.NewConsoleEncoder(encoderConfig())
	case "json":
		ec := encoderConfig()
		ec.EncodeLevel = zapcore.CapitalLevelEncoder
		encoder = zapcore.NewJSONEncoder(ec)
	default:
		encoder = zapcore.NewConsoleEncoder(encoderConfig())
	}

	log = zap.New(zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(sync...), lv),
		zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
	return log, nil
}

type Logger struct {
	lg *zap.SugaredLogger
}

func NewDefault() *Logger {
	var (
		lv  zapcore.Level
		lg  *zap.SugaredLogger
		err error
	)
	if lv, err = zapcore.ParseLevel("info"); err != nil {
		lv = zapcore.InfoLevel
	}
	encoder := zapcore.NewConsoleEncoder(encoderConfig())
	lg = zap.New(zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), lv),
		zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
	return &Logger{lg: lg}
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

func encoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		CallerKey:      "line",
		NameKey:        "logger",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}
