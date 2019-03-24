// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// based on log4go.
package gxlog

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

import (
	gxos "github.com/AlexStocks/goext/os"
	"github.com/AlexStocks/log4go"
)

const (
	logSuffix   = ".log"
	wfLogSuffix = ".wf.log"
)

type Conf struct {
	Name      string // application name
	Dir       string // logger directory
	Level     string // minimum log level
	Console   bool   // whether output log to console
	Daily     bool   // whether rotate log file at mid-night every day
	BackupNum int    // log file backup number. the oldest is deleted.
	BufSize   int    // async logger buffer size
	Json      bool   // whether output json log
}

type Logger struct {
	log4go.Logger
}

// init a logger
func NewLogger(conf Conf) (Logger, error) {
	var (
		err        error
		fileName   string
		logger     log4go.Logger
		fileLogger *log4go.FileLogWriter
	)

	if err = gxos.CreateDir(conf.Dir); err != nil {
		log4go.Error("goext.os.CreateDir(%s) = error{%#v}", conf.Dir, err)
		return Logger{logger}, err
	}

	logger = log4go.NewLogger()
	if conf.Console {
		logger.AddFilter("stdout", logLevel(conf.Level), log4go.NewConsoleLogWriter(conf.Json))
	}

	// create file writer for all log level
	fileName = comLogFileName(conf.Name, conf.Dir, false)
	fileLogger = log4go.NewFileLogWriter(fileName, true, conf.BufSize)
	if fileLogger == nil {
		return Logger{logger}, fmt.Errorf("log4go.NewFileLogWriter(%s) = nil", fileName)
	}
	fileLogger.SetJson(conf.Json)
	fileLogger.SetFormat(log4go.FORMAT_DEFAULT)
	if conf.Daily {
		fileLogger.SetRotateDaily(true)
	}
	if 0 < conf.BackupNum {
		fileLogger.SetRotateMaxBackup(conf.BackupNum)
	}
	logger.AddFilter("log", logLevel(conf.Level), fileLogger)

	// create file writer for warning & fatal & critical
	fileName = comLogFileName(conf.Name, conf.Dir, true)
	fileLogger = log4go.NewFileLogWriter(fileName, true, conf.BufSize)
	if fileLogger == nil {
		return Logger{logger}, fmt.Errorf("log4go.NewFileLogWriter(%s) = nil", fileName)
	}
	fileLogger.SetJson(conf.Json)
	fileLogger.SetFormat(log4go.FORMAT_DEFAULT)
	if conf.Daily {
		fileLogger.SetRotateDaily(true)
	}
	if 0 < conf.BackupNum {
		fileLogger.SetRotateMaxBackup(conf.BackupNum)
	}
	logger.AddFilter("wflog", log4go.WARNING, fileLogger)

	return Logger{logger}, nil
}

func NewLoggerWithConfFile(conf string) Logger {
	logger := log4go.NewLogger()
	return Logger{(&logger).LoadConfiguration(conf)}
}

func comLogFileName(appName string, dir string, err bool) string {
	strings.TrimSuffix(dir, "/")

	if err {
		// log level warning, error, critical
		return filepath.Join(dir, appName+wfLogSuffix)
	}

	return filepath.Join(dir, appName+logSuffix)
}

func logLevel(str string) log4go.Level {
	switch strings.ToUpper(str) {
	case "DEBUG":
		return log4go.DEBUG
	case "TRACE":
		return log4go.TRACE
	case "INFO":
		return log4go.INFO
	case "WARN":
		return log4go.WARNING
	case "ERROR":
		return log4go.ERROR
	case "CRITIC":
		return log4go.CRITICAL
	case "CRITICAL":
		return log4go.CRITICAL
	default:
		return log4go.INFO
	}
}

func funcFileLine() string {
	tm := time.Unix(time.Now().Unix(), 0)
	funcName, file, line, _ := runtime.Caller(3)
	return "[" + tm.Format("2006-01-02/15:04:05 ") +
		runtime.FuncForPC(funcName).Name() +
		": " + filepath.Base(file) +
		": " + fmt.Sprintf("%d", line) +
		"] "
}
