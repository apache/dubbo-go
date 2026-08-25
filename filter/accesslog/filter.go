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

	// drainTimeout bounds how long drainLogs keeps flushing remaining
	// log data while shutting down.
	drainTimeout = 5 * time.Second

	// shutdownWaitTimeout is the maximum time Shutdown waits for the
	// processLogs goroutine to exit, covering the drainTimeout margin.
	shutdownWaitTimeout = 6 * time.Second

	// those fields are the data collected by this filter

	// Types represents the list of argument types in log.
	Types = "types"
	// Arguments represents the arguments string in log.
	Arguments = "arguments"
)

var (
	once            sync.Once
	filterMu        sync.Mutex // guards accessLogFilter against concurrent Shutdown
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
	fileLock     sync.RWMutex // protects fileCache
	fileCache    map[string]*os.File
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownOnce sync.Once
	done         chan struct{} // closed when processLogs exits
}

func newFilter() filter.Filter {
	once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		accessLogFilter = &Filter{
			logChan:   make(chan Data, LogMaxBuffer),
			fileCache: make(map[string]*os.File),
			ctx:       ctx,
			cancel:    cancel,
			done:      make(chan struct{}),
		}
		go accessLogFilter.processLogs()
	})
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
		logger.Warn("[Filter][AccessLog] the channel is full and the access logIntoChannel data will be dropped")
		return
	}
}

// buildAccessLogData builds the access log data
func (f *Filter) buildAccessLogData(_ base.Invoker, invocation base.Invocation) map[string]string {
	dataMap := make(map[string]string, 16)
	attachments := invocation.Attachments()
	itf, ok := stringAttachment(attachments, constant.InterfaceKey)
	if !ok || len(itf) == 0 {
		itf, _ = stringAttachment(attachments, constant.PathKey)
	}
	if itf != "" {
		dataMap[constant.InterfaceKey] = itf
	}
	for _, key := range []string{
		constant.MethodKey,
		constant.VersionKey,
		constant.GroupKey,
		constant.TimestampKey,
		constant.LocalAddr,
		constant.RemoteAddr,
	} {
		if value, ok := stringAttachment(attachments, key); ok {
			dataMap[key] = value
		}
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

func stringAttachment(attachments map[string]any, key string) (string, bool) {
	value, exists := attachments[key]
	if !exists || value == nil {
		return "", false
	}
	stringValue, ok := value.(string)
	if !ok {
		logger.Debugf("[Filter][AccessLog] attachment %q has unexpected type %T and will be omitted", key, value)
		return "", false
	}
	return stringValue, true
}

// OnResponse do nothing
func (f *Filter) OnResponse(_ context.Context, result result.Result, _ base.Invoker, _ base.Invocation) result.Result {
	return result
}

// processLogs runs in a background goroutine to process log data
func (f *Filter) processLogs() {
	// registered first, runs last: signals only after drainLogs completes
	defer close(f.done)
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[Filter][AccessLog] accessLog processLogs panic, err=%v", r)
		}
		f.drainLogs()
	}()

	for {
		select {
		case accessLogData := <-f.logChan:
			f.writeLogToFile(accessLogData)
		case <-f.ctx.Done():
			return
		}
	}
}

// drainLogs drains remaining log data with an absolute deadline
func (f *Filter) drainLogs() {
	deadline := time.Now().Add(drainTimeout)
	for {
		select {
		case accessLogData := <-f.logChan:
			f.writeLogToFile(accessLogData)
		default:
			return
		}
		if time.Now().After(deadline) {
			logger.Warn("[Filter][AccessLog] accessLog drain timeout, some logs may be lost")
			return
		}
	}
}

// writeLogToFile actually write the logs into file
func (f *Filter) writeLogToFile(data Data) {
	accessLog := data.accessLog
	if isDefault(accessLog) {
		logger.Infof("[Filter][AccessLog] %s", data.toLogMessage())
		return
	}

	logFile, err := f.getOrOpenLogFile(accessLog)
	if err != nil {
		logger.Warnf("[Filter][AccessLog] can not open the access log file: %s, %v", accessLog, err)
		return
	}
	logger.Debugf("[Filter][AccessLog] append log to %s", accessLog)
	message := data.toLogMessage()
	message = message + "\n"
	_, err = logFile.WriteString(message)
	if err != nil {
		logger.Warnf("[Filter][AccessLog] can not write the log into access log file, accessLog=%s err=%v", accessLog, err)
	}
}

// needLogRotation checks if the log file needs rotation based on date
func needLogRotation(logFile *os.File) bool {
	now := time.Now().Format(FileDateFormat)
	if fileInfo, err := logFile.Stat(); err == nil {
		last := fileInfo.ModTime().Format(FileDateFormat)
		return now != last
	}
	return true // If we can't stat the file, assume rotation is needed
}

// getOrOpenLogFile gets or opens the log file with proper caching and handle management
func (f *Filter) getOrOpenLogFile(accessLog string) (*os.File, error) {
	f.fileLock.RLock()
	if logFile, exists := f.fileCache[accessLog]; exists {
		// Check if we need to rotate the log
		if !needLogRotation(logFile) {
			f.fileLock.RUnlock()
			return logFile, nil
		}
	}
	f.fileLock.RUnlock()

	// Need to open new file or rotate existing one
	f.fileLock.Lock()
	defer f.fileLock.Unlock()

	// Double-check after acquiring write lock
	if logFile, exists := f.fileCache[accessLog]; exists {
		if !needLogRotation(logFile) {
			return logFile, nil
		}
		// Close the old file before rotation
		if err := logFile.Close(); err != nil {
			logger.Warnf("[Filter][AccessLog] failed to close old log file, accessLog=%s err=%v", accessLog, err)
		}
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
		logger.Warnf("[Filter][AccessLog] can not open the access log file, accessLog=%s err=%v", accessLog, err)
		return nil, err
	}
	now := time.Now().Format(FileDateFormat)
	fileInfo, err := logFile.Stat()
	if err != nil {
		logger.Warnf("[Filter][AccessLog] can not get the info of access log file, accessLog=%s err=%v", accessLog, err)
		if closeErr := logFile.Close(); closeErr != nil {
			logger.Warnf("[Filter][AccessLog] failed to close access log file, accessLog=%s err=%v", accessLog, closeErr)
		}
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
		if closeErr := logFile.Close(); closeErr != nil {
			logger.Warnf("[Filter][AccessLog] failed to close access log file before rotation, accessLog=%s err=%v", accessLog, closeErr)
			return nil, closeErr
		}
		err = os.Rename(accessLog, accessLog+"."+now)
		if err != nil {
			logger.Warnf("[Filter][AccessLog] can not rename access log file, accessLog=%s err=%v", accessLog, err)
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
	filterMu.Lock()
	f := accessLogFilter
	filterMu.Unlock()
	if f != nil {
		f.shutdown()
	}
}

// shutdown gracefully shuts down this filter instance
func (f *Filter) shutdown() {
	f.shutdownOnce.Do(func() {
		// Cancel the context to signal the goroutine to stop. The channel is
		// intentionally left open so concurrent Invoke calls never panic on a
		// closed channel; excess data is dropped once the buffer is full.
		if f.cancel != nil {
			f.cancel()
		}

		// Wait for processLogs to exit (drainLogs timeout is 5s, use 6s margin)
		select {
		case <-f.done:
		case <-time.After(shutdownWaitTimeout):
			logger.Warn("[Filter][AccessLog] shutdown wait for processLogs timeout")
		}

		// Close all cached file handles
		f.fileLock.Lock()
		defer f.fileLock.Unlock()
		for path, file := range f.fileCache {
			if err := file.Close(); err != nil {
				logger.Warnf("[Filter][AccessLog] error closing access log file, path=%s err=%v", path, err)
			}
			delete(f.fileCache, path)
		}
	})
}
