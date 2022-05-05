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

// nolint
package etcdv3

/*
import (
	"reflect"
	"sync"
	"testing"
)

import (
	"github.com/agiledragon/gomonkey"

	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
)

import (
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/etcdv3"
)

type fields struct {
	BaseRegistry   registry.BaseRegistry
	cltLock        sync.Mutex
	client         *gxetcd.Client
	listenerLock   sync.RWMutex
	listener       *etcdv3.EventListener
	dataListener   *dataListener
	configListener *configurationListener
}
type args struct {
	root      string
	node      string
	eventType remoting.Event
}

func newEtcdV3Registry(f fields) *etcdV3Registry {
	return &etcdV3Registry{
		client:         f.client,
		listener:       f.listener,
		dataListener:   f.dataListener,
		configListener: f.configListener,
	}
}

func Test_etcdV3Registry_DoRegister(t *testing.T) {
	var client *gxetcd.Client
	patches := gomonkey.NewPatches()
	patches = patches.ApplyMethod(reflect.TypeOf(client), "RegisterTemp", func(_ *gxetcd.Client, k, v string) error {
		return nil
	})
	defer patches.Reset()

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				client: client,
			},
			args: args{
				root: "/dubbo",
				node: "/go",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newEtcdV3Registry(tt.fields)
			if err := r.DoRegister(tt.args.root, tt.args.node); (err != nil) != tt.wantErr {
				t.Errorf("DoRegister() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_etcdV3Registry_DoUnregister(t *testing.T) {
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "test",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newEtcdV3Registry(tt.fields)
			if err := r.DoUnregister(tt.args.root, tt.args.node); (err != nil) != tt.wantErr {
				t.Errorf("DoUnregister() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

*/
