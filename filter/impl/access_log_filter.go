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

package impl

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
	"os"
	"reflect"
	"strings"
	"time"
)

const (
	//usd in URL.
	AccessLogKey      = "accesslog"
	FileDateFormat    = "2006-01-02"
	MessageDateLayout = "2006-01-02 15:04:05"
	LogMaxBuffer      = 5000
	LogFileMode       = 0600

	// those fields are the data collected by this filter
	Types     = "types"
	Arguments = "arguments"
)

func init() {
	extension.SetFilter(AccessLogKey, GetAccessLogFilter)
}

type AccessLogFilter struct {
	logChan chan AccessLogData
}

func (ef *AccessLogFilter) Invoke(invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	accessLog := invoker.GetUrl().GetParam(AccessLogKey, "")
	logger.Warnf(invoker.GetUrl().String())
	if len(accessLog) > 0 {
		accessLogData := AccessLogData{data: ef.buildAccessLogData(invoker, invocation), accessLog: accessLog}
		ef.logIntoChannel(accessLogData)
	}
	return invoker.Invoke(invocation)
}

// it won't block the invocation
func (ef *AccessLogFilter) logIntoChannel(accessLogData AccessLogData) {
	select {
	case ef.logChan <- accessLogData:
		return
	default:
		logger.Warn("The channel is full and the access logIntoChannel data will be dropped")
		return
	}
}

func (ef *AccessLogFilter) buildAccessLogData(invoker protocol.Invoker, invocation protocol.Invocation) map[string]string {
	dataMap := make(map[string]string)
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
		first := true
		for _, arg := range invocation.Arguments() {
			if first {
				first = false
			} else {
				builder.WriteString(",")
				typeBuilder.WriteString(",")
			}
			builder.WriteString(reflect.ValueOf(arg).String())
			typeBuilder.WriteString(reflect.TypeOf(arg).Name())
		}
		dataMap[Arguments] = builder.String()
		dataMap[Types] = typeBuilder.String()
	}

	return dataMap
}

func (ef *AccessLogFilter) OnResponse(result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func (ef *AccessLogFilter) writeLogToFile(data AccessLogData) {
	accessLog := data.accessLog
	if isDefault(accessLog) {
		logger.Info(data.toLogMessage())
	} else {
		logFile, err := ef.openLogFile(accessLog)
		if err != nil {
			logger.Warnf("Can not open 	the access log file: %s, %v", accessLog, err)
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
}

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
	if now != last {
		err = os.Rename(fileInfo.Name(), fileInfo.Name()+"."+now)
		if err != nil {
			logger.Warnf("Can not rename access log file: %s, %v", fileInfo.Name(), err)
			return nil, err
		} else {
			logFile, err = os.OpenFile(accessLog, os.O_CREATE|os.O_APPEND|os.O_RDWR, LogFileMode)
		}
	}
	return logFile, err
}

func isDefault(accessLog string) bool {
	return strings.EqualFold("true", accessLog) || strings.EqualFold("default", accessLog)
}

func GetAccessLogFilter() filter.Filter {
	accessLogFilter := &AccessLogFilter{logChan: make(chan AccessLogData, LogMaxBuffer)}
	go func() {
		for accessLogData := range accessLogFilter.logChan {
			accessLogFilter.writeLogToFile(accessLogData)
		}
	}()
	return accessLogFilter
}

type AccessLogData struct {
	accessLog string
	data      map[string]string
}

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
