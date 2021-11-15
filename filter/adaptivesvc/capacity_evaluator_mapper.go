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

package adaptivesvc

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/capevaulator"
	"fmt"
	"sync"
)

var (
	capacityEvaluatorMapperSingleton *capacityEvaluatorMapper

	ErrCapEvaluatorNotFound = fmt.Errorf("capacity evaluator not found")
)

func init() {
	capacityEvaluatorMapperSingleton = newCapacityEvaluatorMapper()
}

type capacityEvaluatorMapper struct {
	mutex  *sync.Mutex
	mapper map[string]capevaulator.CapacityEvaluator
}

func newCapacityEvaluatorMapper() *capacityEvaluatorMapper {
	return &capacityEvaluatorMapper{
		mutex:  new(sync.Mutex),
		mapper: make(map[string]capevaulator.CapacityEvaluator),
	}
}

func (m *capacityEvaluatorMapper) setMethodCapacityEvaluator(url *common.URL, methodName string,
	eva capevaulator.CapacityEvaluator) error {
	key := fmt.Sprintf("%s%s", url.Path, methodName)
	m.mutex.Lock()
	capacityEvaluatorMapperSingleton.mapper[key] = eva
	m.mutex.Unlock()
	return nil
}

func (m *capacityEvaluatorMapper) getMethodCapacityEvaluator(url *common.URL, methodName string) (
	capevaulator.CapacityEvaluator, error) {
	key := fmt.Sprintf("%s%s", url.Path, methodName)
	m.mutex.Lock()
	eva, ok := capacityEvaluatorMapperSingleton.mapper[key]
	m.mutex.Unlock()
	if !ok {
		return nil, ErrCapEvaluatorNotFound
	}
	return eva, nil
}
