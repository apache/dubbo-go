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

package hessian

import (
	"log"
)

type logType int

const (
	infoLog logType = iota
	errorLog
)

var _onlyError = false

// showLog 输出日志
func showLog(t logType, format string, v ...interface{}) {
	format = format + "\n"
	if t == errorLog {
		log.Fatalf(format, v...)
		return
	}
	if _onlyError {
		return
	}
	log.Printf(format, v...)
}

func ToggleLog(onlyError bool) {
	_onlyError = onlyError
}
