/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package accesslog providers logging filter.
package accesslog

import (
	"context"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

const (
	// used in URL.
	// FileDateFormat is the date format used for file rotation.
	FileDateFormat = "2006-01-02"
	// MessageDateLayout is the datetime layout used in log message.
	MessageDateLayout = "2006-01-02 15:04:05"
	// LogMaxBuffer is the max buffered log items.
	LogMaxBuffer = 5000
	// LogFileMode is the file permission for access log files.
	LogFileMode = 0o600

	// those fields are the data collected by this filter

	// Types represents the list of argument types in log.
	Types = "types"
	// Arguments represents the arguments string in log.
	Arguments = "arguments"
)

var (
	once            sync.Once
	accessLogFilter *Filter
)

func init() {
	extension.SetFilter(constant.AccessLogFilterKey, newFilter)
}

// Filter for Access Log
/**
 * Although the access log filter is a default filter,
 * you should config "accesslog" in service's config to tell the filter where store the access log.
 * for example:
 * "UserProvider":
 *   registry: "hangzhouzk"
 *   protocol : "dubbo"
 *   interface : "com.ikurento.user.UserProvider"
 *   ... # other configuration
 *   accesslog: "/your/path/to/store/the/log/", # it should be the path of file.
 *
 * the value of "accesslog" can be "true" or "default" too.
 * If the value is one of them, the access log will be record in log file which defined in log.yml
 * AccessLogFilter is designed to be singleton
 */
type Filter struct {
	logChan      chan Data
	fileCache    map[string]*os.File
	fileLock     sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownOnce sync.Once
}

func newFilter() filter.Filter {
	if accessLogFilter == nil {
		once.Do(func() {
			ctx, cancel := context.WithCancel(context.Background())
			accessLogFilter = &Filter{
				logChan:   make(chan Data, LogMaxBuffer),
				fileCache: make(map[string]*os.File),
				ctx:       ctx,
				cancel:    cancel,
			}
			accessLogFilter.start()
		})
	}
	return accessLogFilter
}

// Invoke will check whether user wants to use this filter.
// If we find the value of key constant.AccessLogFilterKey, we will log the invocation info
func (f *Filter) Invoke(ctx context.Context, invoker base.Invoker, invocation base.Invocation) result.Result {
	accessLog := invoker.GetURL().GetParam(constant.AccessLogFilterKey, "")

	// the user do not
	if len(accessLog) > 0 {
		accessLogData := Data{data: f.buildAccessLogData(invoker, invocation), accessLog: accessLog}
		f.logIntoChannel(accessLogData)
	}
	return invoker.Invoke(ctx, invocation)
}

// logIntoChannel won't block the invocation
func (f *Filter) logIntoChannel(accessLogData Data) {
	select {
	case f.logChan <- accessLogData:
		return
	default:
		logger.Warn("The channel is full and the access logIntoChannel data will be dropped")
		return
	}
}

// buildAccessLogData builds the access log data
func (f *Filter) buildAccessLogData(_ base.Invoker, invocation base.Invocation) map[string]string {
	dataMap := make(map[string]string, 16)
	attachments := invocation.Attachments()
	itf := attachments[constant.InterfaceKey]
	if itf == nil || len(itf.(string)) == 0 {
		itf = attachments[constant.PathKey]
	}
	if itf != nil {
		dataMap[constant.InterfaceKey] = itf.(string)
	}
	if v, ok := attachments[constant.MethodKey]; ok && v != nil {
		dataMap[constant.MethodKey] = v.(string)
	}
	if v, ok := attachments[constant.VersionKey]; ok && v != nil {
		dataMap[constant.VersionKey] = v.(string)
	}
	if v, ok := attachments[constant.GroupKey]; ok && v != nil {
		dataMap[constant.GroupKey] = v.(string)
	}
	if v, ok := attachments[constant.TimestampKey]; ok && v != nil {
		dataMap[constant.TimestampKey] = v.(string)
	}
	if v, ok := attachments[constant.LocalAddr]; ok && v != nil {
		dataMap[constant.LocalAddr] = v.(string)
	}
	if v, ok := attachments[constant.RemoteAddr]; ok && v != nil {
		dataMap[constant.RemoteAddr] = v.(string)
	}

	if len(invocation.Arguments()) > 0 {
		builder := strings.Builder{}
		// todo(after the paramTypes were set to the invocation. we should change this implementation)
		typeBuilder := strings.Builder{}

		builder.WriteString(reflect.ValueOf(invocation.Arguments()[0]).String())
		typeBuilder.WriteString(reflect.TypeOf(invocation.Arguments()[0]).Name())
		for idx := 1; idx < len(invocation.Arguments()); idx++ {
			arg := invocation.Arguments()[idx]
			builder.WriteString(",")
			builder.WriteString(reflect.ValueOf(arg).String())

			typeBuilder.WriteString(",")
			typeBuilder.WriteString(reflect.TypeOf(arg).Name())
		}
		dataMap[Arguments] = builder.String()
		dataMap[Types] = typeBuilder.String()
	}

	return dataMap
}

// OnResponse do nothing
func (f *Filter) OnResponse(_ context.Context, result result.Result, _ base.Invoker, _ base.Invocation) result.Result {
	return result
}

// start initializes and starts the background goroutine for processing logs
func (f *Filter) start() {
	go f.processLogs()
}

// processLogs runs in a background goroutine to process log data
func (f *Filter) processLogs() {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("AccessLog processLogs panic: %v", r)
		}
		f.drainLogs()
	}()

	for {
		select {
		case accessLogData, ok := <-f.logChan:
			if !ok {
				return
			}
			f.writeLogToFileWithTimeout(accessLogData, 5*time.Second)
		case <-f.ctx.Done():
			return
		}
	}
}

// drainLogs drains remaining log data with timeout protection
func (f *Filter) drainLogs() {
	timeout := time.After(5 * time.Second)
	for {
		select {
		case accessLogData, ok := <-f.logChan:
			if !ok {
				return
			}
			f.writeLogToFileWithTimeout(accessLogData, 1*time.Second)
		case <-timeout:
			logger.Warnf("AccessLog drain timeout, some logs may be lost")
			return
		default:
			return
		}
	}
}

// writeLogToFileWithTimeout writes log with timeout protection
func (f *Filter) writeLogToFileWithTimeout(data Data, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		f.writeLogToFile(data)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(timeout):
		logger.Warnf("AccessLog writeLogToFile timeout for: %s", data.accessLog)
	}
}

// writeLogToFile actually write the logs into file
func (f *Filter) writeLogToFile(data Data) {
	accessLog := data.accessLog
	if isDefault(accessLog) {
		logger.Info(data.toLogMessage())
		return
	}

	logFile, err := f.getOrOpenLogFile(accessLog)
	if err != nil {
		logger.Warnf("Can not open the access log file: %s, %v", accessLog, err)
		return
	}
	logger.Debugf("Append log to %s", accessLog)
	message := data.toLogMessage()
	message = message + "\n"
	_, err = logFile.WriteString(message)
	if err != nil {
		logger.Warnf("Can not write the log into access log file: %s, %v", accessLog, err)
	}
}

// getOrOpenLogFile gets or opens the log file with proper caching and handle management
func (f *Filter) getOrOpenLogFile(accessLog string) (*os.File, error) {
	f.fileLock.RLock()
	if logFile, exists := f.fileCache[accessLog]; exists {
		// Check if we need to rotate the log
		now := time.Now().Format(FileDateFormat)
		if fileInfo, err := logFile.Stat(); err == nil {
			last := fileInfo.ModTime().Format(FileDateFormat)
			if now == last {
				f.fileLock.RUnlock()
				return logFile, nil
			}
		}
	}
	f.fileLock.RUnlock()

	// Need to open new file or rotate existing one
	f.fileLock.Lock()
	defer f.fileLock.Unlock()

	// Double-check after acquiring write lock
	if logFile, exists := f.fileCache[accessLog]; exists {
		now := time.Now().Format(FileDateFormat)
		if fileInfo, err := logFile.Stat(); err == nil {
			last := fileInfo.ModTime().Format(FileDateFormat)
			if now == last {
				return logFile, nil
			}
		}
		// Close the old file before rotation
		logFile.Close()
		delete(f.fileCache, accessLog)
	}

	logFile, err := f.openLogFile(accessLog)
	if err != nil {
		return nil, err
	}

	f.fileCache[accessLog] = logFile
	return logFile, nil
}

// openLogFile will open the log file with append mode.
// If the file is not found, it will create the file.
// Actually, the accessLog is the filename
func (f *Filter) openLogFile(accessLog string) (*os.File, error) {
	logFile, err := os.OpenFile(accessLog, os.O_CREATE|os.O_APPEND|os.O_RDWR, LogFileMode)
	if err != nil {
		logger.Warnf("Can not open the access log file: %s, %v", accessLog, err)
		return nil, err
	}
	now := time.Now().Format(FileDateFormat)
	fileInfo, err := logFile.Stat()
	if err != nil {
		logger.Warnf("Can not get the info of access log file: %s, %v", accessLog, err)
		return nil, err
	}
	last := fileInfo.ModTime().Format(FileDateFormat)

	// this is confused.
	// for example, if the last = '2020-03-04'
	// and today is '2020-03-05'
	// we will create one new file to log access data
	// By this way, we can split the access log based on days.
	// use 'accessLog' as complete path to avoid log not found.
	if now != last {
		err = os.Rename(accessLog, accessLog+"."+now)
		if err != nil {
			logger.Warnf("Can not rename access log file: %s, %v", accessLog, err)
			return nil, err
		}
		logFile, err = os.OpenFile(accessLog, os.O_CREATE|os.O_APPEND|os.O_RDWR, LogFileMode)
	}
	return logFile, err
}

// isDefault check whether accessLog == true or accessLog == default
func isDefault(accessLog string) bool {
	return strings.EqualFold("true", accessLog) || strings.EqualFold("default", accessLog)
}

// Data defines the data that will be log into file
type Data struct {
	accessLog string
	data      map[string]string
}

// toLogMessage convert the Data to String
func (d *Data) toLogMessage() string {
	builder := strings.Builder{}
	builder.WriteString("[")
	builder.WriteString(d.data[constant.TimestampKey])
	builder.WriteString("] ")
	builder.WriteString(d.data[constant.RemoteAddr])
	builder.WriteString(" -> ")
	builder.WriteString(d.data[constant.LocalAddr])
	builder.WriteString(" - ")
	if len(d.data[constant.GroupKey]) > 0 {
		builder.WriteString(d.data[constant.GroupKey])
		builder.WriteString("/")
	}

	builder.WriteString(d.data[constant.InterfaceKey])

	if len(d.data[constant.VersionKey]) > 0 {
		builder.WriteString(":")
		builder.WriteString(d.data[constant.VersionKey])
	}

	builder.WriteString(" ")
	builder.WriteString(d.data[constant.MethodKey])
	builder.WriteString("(")
	if len(d.data[Types]) > 0 {
		builder.WriteString(d.data[Types])
	}
	builder.WriteString(") ")

	if len(d.data[Arguments]) > 0 {
		builder.WriteString(d.data[Arguments])
	}
	return builder.String()
}

// Shutdown gracefully shuts down the access log filter
// This should be called during application shutdown to prevent goroutine leaks
func Shutdown() {
	if accessLogFilter != nil {
		accessLogFilter.shutdown()
	}
}

// shutdown gracefully shuts down this filter instance
func (f *Filter) shutdown() {
	f.shutdownOnce.Do(func() {
		// Cancel the context to signal goroutine to stop
		if f.cancel != nil {
			f.cancel()
		}

		// Close the channel to stop accepting new logs
		if f.logChan != nil {
			close(f.logChan)
		}

		// Close all cached file handles
		f.fileLock.Lock()
		for path, file := range f.fileCache {
			if err := file.Close(); err != nil {
				logger.Warnf("Error closing access log file %s: %v", path, err)
			}
			delete(f.fileCache, path)
		}
		f.fileLock.Unlock()
	})
}
