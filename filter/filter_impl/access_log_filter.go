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

package filter_impl

import (
	"context"
	"os"
	"reflect"
	"strings"
	"time"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
)

const (
	//used in URL.

	// nolint
	FileDateFormat = "2006-01-02"
	// nolint
	MessageDateLayout = "2006-01-02 15:04:05"
	// nolint
	LogMaxBuffer = 5000
	// nolint
	LogFileMode = 0600

	// those fields are the data collected by this filter

	// nolint
	Types = "types"
	// nolint
	Arguments = "arguments"
)

func init() {
	extension.SetFilter(constant.ACCESS_LOG_KEY, GetAccessLogFilter)
}

/*
 * AccessLogFilter
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
type AccessLogFilter struct {
	logChan chan AccessLogData
}

// Invoke will check whether user wants to use this filter.
// If we find the value of key constant.ACCESS_LOG_KEY, we will log the invocation info
func (ef *AccessLogFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	accessLog := invoker.GetUrl().GetParam(constant.ACCESS_LOG_KEY, "")

	// the user do not
	if len(accessLog) > 0 {
		accessLogData := AccessLogData{data: ef.buildAccessLogData(invoker, invocation), accessLog: accessLog}
		ef.logIntoChannel(accessLogData)
	}
	return invoker.Invoke(ctx, invocation)
}

// logIntoChannel won't block the invocation
func (ef *AccessLogFilter) logIntoChannel(accessLogData AccessLogData) {
	select {
	case ef.logChan <- accessLogData:
		return
	default:
		logger.Warn("The channel is full and the access logIntoChannel data will be dropped")
		return
	}
}

// buildAccessLogData builds the access log data
func (ef *AccessLogFilter) buildAccessLogData(_ protocol.Invoker, invocation protocol.Invocation) map[string]string {
	dataMap := make(map[string]string, 16)
	attachments := invocation.Attachments()
	dataMap[constant.INTERFACE_KEY] = attachments[constant.INTERFACE_KEY]
	dataMap[constant.METHOD_KEY] = invocation.MethodName()
	dataMap[constant.VERSION_KEY] = attachments[constant.VERSION_KEY]
	dataMap[constant.GROUP_KEY] = attachments[constant.GROUP_KEY]
	dataMap[constant.TIMESTAMP_KEY] = time.Now().Format(MessageDateLayout)
	dataMap[constant.LOCAL_ADDR], _ = attachments[constant.LOCAL_ADDR]
	dataMap[constant.REMOTE_ADDR], _ = attachments[constant.REMOTE_ADDR]

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
func (ef *AccessLogFilter) OnResponse(_ context.Context, result protocol.Result, _ protocol.Invoker, _ protocol.Invocation) protocol.Result {
	return result
}

// writeLogToFile actually write the logs into file
func (ef *AccessLogFilter) writeLogToFile(data AccessLogData) {
	accessLog := data.accessLog
	if isDefault(accessLog) {
		logger.Info(data.toLogMessage())
		return
	}

	logFile, err := ef.openLogFile(accessLog)
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

// openLogFile will open the log file with append mode.
// If the file is not found, it will create the file.
// Actually, the accessLog is the filename
// You may find out that, once we want to write access log into log file,
// we open the file again and again.
// It needs to be optimized.
func (ef *AccessLogFilter) openLogFile(accessLog string) (*os.File, error) {
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
	if now != last {
		err = os.Rename(fileInfo.Name(), fileInfo.Name()+"."+now)
		if err != nil {
			logger.Warnf("Can not rename access log file: %s, %v", fileInfo.Name(), err)
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

// GetAccessLogFilter return the instance of AccessLogFilter
func GetAccessLogFilter() filter.Filter {
	accessLogFilter := &AccessLogFilter{logChan: make(chan AccessLogData, LogMaxBuffer)}
	go func() {
		for accessLogData := range accessLogFilter.logChan {
			accessLogFilter.writeLogToFile(accessLogData)
		}
	}()
	return accessLogFilter
}

// AccessLogData defines the data that will be log into file
type AccessLogData struct {
	accessLog string
	data      map[string]string
}

// toLogMessage convert the AccessLogData to String
func (ef *AccessLogData) toLogMessage() string {
	builder := strings.Builder{}
	builder.WriteString("[")
	builder.WriteString(ef.data[constant.TIMESTAMP_KEY])
	builder.WriteString("] ")
	builder.WriteString(ef.data[constant.REMOTE_ADDR])
	builder.WriteString(" -> ")
	builder.WriteString(ef.data[constant.LOCAL_ADDR])
	builder.WriteString(" - ")
	if len(ef.data[constant.GROUP_KEY]) > 0 {
		builder.WriteString(ef.data[constant.GROUP_KEY])
		builder.WriteString("/")
	}

	builder.WriteString(ef.data[constant.INTERFACE_KEY])

	if len(ef.data[constant.VERSION_KEY]) > 0 {
		builder.WriteString(":")
		builder.WriteString(ef.data[constant.VERSION_KEY])
	}

	builder.WriteString(" ")
	builder.WriteString(ef.data[constant.METHOD_KEY])
	builder.WriteString("(")
	if len(ef.data[Types]) > 0 {
		builder.WriteString(ef.data[Types])
	}
	builder.WriteString(") ")

	if len(ef.data[Arguments]) > 0 {
		builder.WriteString(ef.data[Arguments])
	}
	return builder.String()
}
