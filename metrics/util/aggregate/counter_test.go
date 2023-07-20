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
	"testing"
	"time"
)

func TestTimeWindowCounterCount(t1 *testing.T) {
	tests := []struct {
		name       string
		queryTimes int
		want       float64
	}{
		{
			name:       "Query Times: 0",
			queryTimes: 0,
			want:       0,
		},
		{
			name:       "Query Times: 3",
			queryTimes: 3,
			want:       3,
		},
		{
			name:       "Query Times: 10",
			queryTimes: 10,
			want:       10,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := NewTimeWindowCounter(10, 1)
			for i := 0; i < tt.queryTimes; i++ {
				if i%3 == 0 {
					time.Sleep(time.Millisecond * 100)
				}
				t.Inc()
			}
			if got := t.Count(); got != tt.want {
				t1.Errorf("Count() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeWindowCounterLivedSeconds(t1 *testing.T) {
	tests := []struct {
		name       string
		queryTimes int
		want       int64
	}{
		{
			name:       "Query Times: 0",
			queryTimes: 0,
			want:       0,
		},
		{
			name:       "Query Times: 3",
			queryTimes: 3,
			want:       1,
		},
		{
			name:       "Query Times: 9",
			queryTimes: 9,
			want:       3,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := NewTimeWindowCounter(10, 10)
			for i := 0; i < tt.queryTimes; i++ {
				if i%3 == 0 {
					time.Sleep(time.Second * 1)
				}
				t.Inc()
			}
			if got := t.LivedSeconds(); got != tt.want {
				t1.Errorf("Count() = %v, want %v", got, tt.want)
			}
		})
	}
}
