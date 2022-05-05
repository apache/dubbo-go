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

package etcdv3

import (
	"reflect"
	"testing"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
)

type args struct {
	client *gxetcd.Client
}

func TestNewEventListener(t *testing.T) {
	tests := []struct {
		name string
		args args
		want *EventListener
	}{
		{
			name: "test",
			args: args{
				client: nil,
			},
			want: NewEventListener(nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEventListener(tt.args.client); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEventListener() = %v, want %v", got, tt.want)
			}
		})
	}
}
