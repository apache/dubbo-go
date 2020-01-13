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

package kubernetes

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestStore(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	s := newStore(ctx)

	wg := sync.WaitGroup{}

	for i := 0; i < 2; i++ {

		wg.Add(1)

		go func() {
			defer wg.Done()
			w, err := s.Watch("key-1", false)
			if err != nil {
				fmt.Println("watch spec result", err)
				return
			}
			for e := range w.ResultChan() {
				fmt.Printf("consumer %s got %s\n", w.ID(), e.Key)
			}
		}()
	}
	for i := 2; i < 3; i++ {

		wg.Add(1)
		go func() {

			defer wg.Done()
			w, err := s.Watch("key", true)
			if err != nil {
				fmt.Println("watch prefix result", err)
				return
			}
			for e := range w.ResultChan() {
				fmt.Printf("prefix consumer %s got %s\n", w.ID(), e.Key)
			}
		}()
	}

	for i := 0; i < 5; i++ {
		go func(i int) {
			if err := s.Put(&Object{
				Key:   "key-" + strconv.Itoa(i),
				Value: strconv.Itoa(i),
			}); err != nil {
				t.Fatal(err)
			}
		}(i)
	}

	wg.Wait()
}
