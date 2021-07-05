package logger

import (
	"io/ioutil"
	"path"
)

import (
	"github.com/apache/dubbo-getty"
	perrors "github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

var baseZapLevel zap.AtomicLevel
var baseZapLogger *zap.Logger

func InitZapLog(logConfFile string) error {
	if logConfFile == "" {
		InitZapLogger(nil)
		return perrors.New("log configure file name is nil")
	}
	if path.Ext(logConfFile) != ".yml" {
		InitZapLogger(nil)
		return perrors.Errorf("log configure file name{%s} suffix must be .yml", logConfFile)
	}

	confFileStream, err := ioutil.ReadFile(logConfFile)
	if err != nil {
		InitZapLogger(nil)
		return perrors.Errorf("ioutil.ReadFile(file:%s) = error:%v", logConfFile, err)
	}

	conf := &zap.Config{}
	err = yaml.Unmarshal(confFileStream, conf)
	if err != nil {
		InitZapLogger(nil)
		return perrors.Errorf("[Unmarshal]init logger error: %v", err)
	}

	InitZapLogger(conf)

	return nil
}

func InitZapLogger(conf *zap.Config) {
	var zapLoggerConfig zap.Config
	if conf == nil {
		zapLoggerEncoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
		zapLoggerConfig = zap.Config{
			Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
			Development:      false,
			Encoding:         "console",
			EncoderConfig:    zapLoggerEncoderConfig,
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		}
	} else {
		zapLoggerConfig = *conf
	}
	baseZapLogger, _ = zapLoggerConfig.Build(zap.AddCallerSkip(1))
	baseZapLevel = zapLoggerConfig.Level
	logger = &DubboLogger{Logger: baseZapLogger.Sugar(), dynamicLevel: baseZapLevel.String(), loggerName: "zap"}

	// set getty log
	getty.SetLogger(logger)
}

func SetZapLevel(level string) {
	l := new(zapcore.Level)
	if err := l.Set(level); err == nil {
		baseZapLevel.SetLevel(*l)
	}
}
