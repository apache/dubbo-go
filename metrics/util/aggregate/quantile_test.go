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

import "testing"

func TestTimeWindowQuantile_AddAndQuantile(t1 *testing.T) {
	timeWindowQuantile := NewTimeWindowQuantile(100, 10, 1)
	for i := 1; i <= 100; i++ {
		timeWindowQuantile.Add(float64(i))
	}

	type args struct {
		q float64
	}

	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "Quantile: 0.01",
			args: args{
				q: 0.01,
			},
			want: 1.5,
		},
		{
			name: "Quantile: 0.99",
			args: args{
				q: 0.99,
			},
			want: 99.5,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := timeWindowQuantile
			if got := t.Quantile(tt.args.q); got != tt.want {
				t1.Errorf("Quantile() = %v, want %v", got, tt.want)
			}
		})
	}
}
