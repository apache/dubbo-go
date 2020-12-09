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
)

type Serializer interface {
	Marshal(p DubboPackage) ([]byte, error)
	Unmarshal([]byte, *DubboPackage) error
}

func LoadSerializer(p *DubboPackage) error {
	// NOTE: default serialID is S_Hessian
	serialID := p.Header.SerialID
	if serialID == 0 {
		serialID = constant.S_Hessian2
	}
	serializer, err := GetSerializerById(serialID)
	if err != nil {
		panic(err)
	}
	p.SetSerializer(serializer.(Serializer))
	return nil
}
