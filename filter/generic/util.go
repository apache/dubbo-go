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

package generic

import (
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/filter/generic/generalizer"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// isCallingToGenericService check if it calls to a generic service
func isCallingToGenericService(invoker protocol.Invoker, invocation protocol.Invocation) bool {
	return isGeneric(invoker.GetURL().GetParam(constant.GenericKey, "")) &&
		invocation.MethodName() != constant.Generic &&
		invocation.MethodName() != constant.GenericAsync
}

// isMakingAGenericCall check if it is making a generic call to a generic service
func isMakingAGenericCall(invoker protocol.Invoker, invocation protocol.Invocation) bool {
	return isGeneric(invoker.GetURL().GetParam(constant.GenericKey, "")) &&
		(invocation.MethodName() == constant.Generic ||
			invocation.MethodName() == constant.GenericAsync) &&
		invocation.Arguments() != nil &&
		len(invocation.Arguments()) == 3
}

// isGeneric receives a generic field from url of invoker to determine whether the service is generic or not
func isGeneric(generic string) bool {
	lowerGeneric := strings.ToLower(generic)
	return lowerGeneric == constant.GenericSerializationDefault
}

// toUnexport is to lower the first letter
func toUnexport(a string) string {
	return strings.ToLower(a[:1]) + a[1:]
}

// toExport is to upper the first letter
func toExport(a string) string {
	return strings.ToUpper(a[:1]) + a[1:]
}

func getGeneralizer(generic string) (g generalizer.Generalizer) {
	switch strings.ToLower(generic) {
	case constant.GenericSerializationDefault:
		g = generalizer.GetMapGeneralizer()
	case constant.GenericSerializationGson:
		g = generalizer.GetGsonGeneralizer()

	default:
		logger.Debugf("\"%s\" is not supported, use the default generalizer(MapGeneralizer)", generic)
		g = generalizer.GetMapGeneralizer()
	}
	return
}
