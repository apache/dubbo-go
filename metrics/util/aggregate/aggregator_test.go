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

package aggregate

import (
	"math/rand"
	"reflect"
	"sync"
	"testing"
)

func TestTimeWindowAggregatorAddAndResult(t *testing.T) {
	timeWindowAggregator := NewTimeWindowAggregator(10, 1)
	timeWindowAggregator.Add(10)
	timeWindowAggregator.Add(20)
	timeWindowAggregator.Add(30)

	tests := []struct {
		name string
		want *Result
	}{
		{
			name: "Result",
			want: &Result{
				Total: 60,
				Min:   10,
				Max:   30,
				Avg:   20,
				Count: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeWindowAggregator.Result()
			got.Last = 0 // NaN can not equal
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Result() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkTimeWindowAggregatorAdd(b *testing.B) {
	wg := sync.WaitGroup{}
	tw := NewTimeWindowAggregator(10, 1)
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tw.Add(rand.Float64() * 100)
		}()
	}
	wg.Wait()
}

func BenchmarkTimeWindowAggregatorResult(b *testing.B) {
	wg := sync.WaitGroup{}
	tw := NewTimeWindowAggregator(10, 1)
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			tw.Add(rand.Float64() * 100)
		}()
		go func() {
			defer wg.Done()
			tw.Result()
		}()
	}
	wg.Wait()
}
