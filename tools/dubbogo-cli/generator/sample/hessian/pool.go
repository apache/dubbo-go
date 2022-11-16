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
	"sync"
)

// Task 任务
type Task interface {
	Execute()
}

// Pool 线程池
type Pool struct {
	wg *sync.WaitGroup

	taskQueue chan Task
}

func NewPool(max int) *Pool {
	return &Pool{
		wg:        &sync.WaitGroup{},
		taskQueue: make(chan Task, max),
	}
}

func (p Pool) Execute(t Task) {
	p.wg.Add(1)
	p.taskQueue <- t
	go func() {
		t.Execute()
		<-p.taskQueue
		p.wg.Done()
	}()
}

func (p Pool) Wait() {
	p.wg.Wait()
}
