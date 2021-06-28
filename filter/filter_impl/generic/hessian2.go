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
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
)

const (
	// same with dubbo
	Hessian2 = "true"
)

func init() {
	extension.SetGenericProcessor(Hessian2, NewHessian2GenericProcessor())
}

type hessian2GenericProcessor struct{}

func NewHessian2GenericProcessor() filter.GenericProcessor {
	return &hessian2GenericProcessor{}
}

func (hessian2GenericProcessor) Serialize(args interface{}) (interface{}, error) {
	panic("implement me")
}

func (hessian2GenericProcessor) Deserialize(args interface{}) (interface{}, error) {
	panic("implement me")
}
