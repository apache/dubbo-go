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
	"fmt"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter"
)

var (
	limiterMapperSingleton *limiterMapper

	ErrLimiterNotFoundOnMapper = fmt.Errorf("limiter not found on mapper")
	ErrLimiterTypeNotFound     = fmt.Errorf("limiter type not found")
)

func init() {
	limiterMapperSingleton = newLimiterMapper()
}

type limiterMapper struct {
	rwMutex *sync.RWMutex
	mapper  map[string]limiter.Limiter
}

func newLimiterMapper() *limiterMapper {
	return &limiterMapper{
		rwMutex: new(sync.RWMutex),
		mapper:  make(map[string]limiter.Limiter),
	}
}

func (m *limiterMapper) newAndSetMethodLimiter(url *common.URL, methodName string, limiterType int) (limiter.Limiter, error) {
	key := fmt.Sprintf("%s%s", url.Path, methodName)

	var (
		l  limiter.Limiter
		ok bool
	)
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()

	if l, ok = limiterMapperSingleton.mapper[key]; ok {
		return l, nil
	}
	switch limiterType {
	case limiter.HillClimbingLimiter:
		l = limiter.NewHillClimbing()
	default:
		return nil, ErrLimiterTypeNotFound
	}
	limiterMapperSingleton.mapper[key] = l
	return l, nil
}

func (m *limiterMapper) getMethodLimiter(url *common.URL, methodName string) (
	limiter.Limiter, error) {
	key := fmt.Sprintf("%s%s", url.Path, methodName)
	m.rwMutex.RLock()
	l, ok := limiterMapperSingleton.mapper[key]
	m.rwMutex.RUnlock()
	if !ok {
		return nil, ErrLimiterNotFoundOnMapper
	}
	return l, nil
}
