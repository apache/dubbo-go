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
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/coreos/etcd/embed"
	"github.com/coreos/etcd/mvcc/mvccpb"
	perrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/connectivity"
)

const defaultEtcdV3WorkDir = "/tmp/default-dubbo-go-remote.etcd"

// tests dataset
var tests = []struct {
	input struct {
		k string
		v string
	}
}{
	{input: struct {
		k string
		v string
	}{k: "name", v: "scott.wang"}},
	{input: struct {
		k string
		v string
	}{k: "namePrefix", v: "prefix.scott.wang"}},
	{input: struct {
		k string
		v string
	}{k: "namePrefix1", v: "prefix1.scott.wang"}},
	{input: struct {
		k string
		v string
	}{k: "age", v: "27"}},
}

// test dataset prefix
const prefix = "name"

type ClientTestSuite struct {
	suite.Suite

	etcdConfig struct {
		name      string
		endpoints []string
		timeout   time.Duration
		heartbeat int
	}

	etcd *embed.Etcd

	client *Client
}

// start etcd server
func (suite *ClientTestSuite) SetupSuite() {

	t := suite.T()

	DefaultListenPeerURLs := "http://localhost:2382"
	DefaultListenClientURLs := "http://localhost:2381"
	lpurl, _ := url.Parse(DefaultListenPeerURLs)
	lcurl, _ := url.Parse(DefaultListenClientURLs)
	cfg := embed.NewConfig()
	cfg.LPUrls = []url.URL{*lpurl}
	cfg.LCUrls = []url.URL{*lcurl}
	cfg.Dir = defaultEtcdV3WorkDir
	e, err := embed.StartEtcd(cfg)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-e.Server.ReadyNotify():
		t.Log("Server is ready!")
	case <-time.After(60 * time.Second):
		e.Server.Stop() // trigger a shutdown
		t.Logf("Server took too long to start!")
	}

	suite.etcd = e
	return
}

// stop etcd server
func (suite *ClientTestSuite) TearDownSuite() {
	suite.etcd.Close()
	if err := os.RemoveAll(defaultEtcdV3WorkDir); err != nil {
		suite.FailNow(err.Error())
	}
}

func (suite *ClientTestSuite) setUpClient() *Client {
	c, err := NewClient(suite.etcdConfig.name,
		suite.etcdConfig.endpoints,
		suite.etcdConfig.timeout,
		suite.etcdConfig.heartbeat)
	if err != nil {
		suite.T().Fatal(err)
	}
	return c
}

// set up a client for suite
func (suite *ClientTestSuite) SetupTest() {
	c := suite.setUpClient()
	c.CleanKV()
	suite.client = c
	return
}

func (suite *ClientTestSuite) TestClientClose() {

	c := suite.client
	t := suite.T()

	defer c.Close()
	if c.rawClient.ActiveConnection().GetState() != connectivity.Ready {
		t.Fatal(suite.client.rawClient.ActiveConnection().GetState())
	}
}

func (suite *ClientTestSuite) TestClientValid() {

	c := suite.client
	t := suite.T()

	if !c.Valid() {
		t.Fatal("client is not valid")
	}
	c.Close()
	if suite.client.Valid() != false {
		t.Fatal("client is valid")
	}
}

func (suite *ClientTestSuite) TestClientDone() {

	c := suite.client

	go func() {
		time.Sleep(2 * time.Second)
		c.Close()
	}()

	c.Wait.Wait()

	if c.Valid() {
		suite.T().Fatal("client should be invalid then")
	}
}

func (suite *ClientTestSuite) TestClientCreateKV() {

	tests := tests

	c := suite.client
	t := suite.T()

	defer suite.client.Close()

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v
		expect := tc.input.v

		if err := c.Create(k, v); err != nil {
			t.Fatal(err)
		}

		value, err := c.Get(k)
		if err != nil {
			t.Fatal(err)
		}

		if value != expect {
			t.Fatalf("expect %v but get %v", expect, value)
		}

	}
}

func (suite *ClientTestSuite) TestClientDeleteKV() {

	tests := tests
	c := suite.client
	t := suite.T()

	defer c.Close()

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v
		expect := ErrKVPairNotFound

		if err := c.Create(k, v); err != nil {
			t.Fatal(err)
		}

		if err := c.Delete(k); err != nil {
			t.Fatal(err)
		}

		_, err := c.Get(k)
		if perrors.Cause(err) == expect {
			continue
		}

		if err != nil {
			t.Fatal(err)
		}
	}

}

func (suite *ClientTestSuite) TestClientGetChildrenKVList() {

	tests := tests

	c := suite.client
	t := suite.T()

	var expectKList []string
	var expectVList []string

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v

		if strings.Contains(k, prefix) {
			expectKList = append(expectKList, k)
			expectVList = append(expectVList, v)
		}

		if err := c.Create(k, v); err != nil {
			t.Fatal(err)
		}
	}

	kList, vList, err := c.GetChildrenKVList(prefix)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(expectKList, kList) && reflect.DeepEqual(expectVList, vList) {
		return
	}

	t.Fatalf("expect keylist %v but got %v expect valueList %v but got %v ", expectKList, kList, expectVList, vList)

}

func (suite *ClientTestSuite) TestClientWatch() {

	tests := tests

	c := suite.client
	t := suite.T()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {

		defer wg.Done()

		wc, err := c.watch(prefix)
		if err != nil {
			t.Fatal(err)
		}

		events := make([]mvccpb.Event, 0)
		var eCreate, eDelete mvccpb.Event

		for e := range wc {

			for _, event := range e.Events {
				events = append(events, (mvccpb.Event)(*event))
				if event.Type == mvccpb.PUT {
					eCreate = (mvccpb.Event)(*event)
				}
				if event.Type == mvccpb.DELETE {
					eDelete = (mvccpb.Event)(*event)
				}
				t.Logf("type IsCreate %v k %s v %s", event.IsCreate(), event.Kv.Key, event.Kv.Value)
			}
		}

		assert.Equal(t, 2, len(events))
		assert.Contains(t, events, eCreate)
		assert.Contains(t, events, eDelete)
	}()

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v

		if err := c.Create(k, v); err != nil {
			t.Fatal(err)
		}

		if err := c.delete(k); err != nil {
			t.Fatal(err)
		}
	}

	c.Close()

	wg.Wait()

}

func (suite *ClientTestSuite) TestClientRegisterTemp() {

	c := suite.client
	observeC := suite.setUpClient()
	t := suite.T()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		completePath := path.Join("scott", "wang")
		wc, err := observeC.watch(completePath)
		if err != nil {
			t.Fatal(err)
		}

		events := make([]mvccpb.Event, 0)
		var eCreate, eDelete mvccpb.Event

		for e := range wc {

			for _, event := range e.Events {
				events = append(events, (mvccpb.Event)(*event))
				if event.Type == mvccpb.DELETE {
					eDelete = (mvccpb.Event)(*event)
					t.Logf("complete key (%s) is delete", completePath)
					observeC.Close()
					break
				}
				eCreate = (mvccpb.Event)(*event)
				t.Logf("type IsCreate %v k %s v %s", event.IsCreate(), event.Kv.Key, event.Kv.Value)
			}
		}

		assert.Equal(t, 2, len(events))
		assert.Contains(t, events, eCreate)
		assert.Contains(t, events, eDelete)
	}()

	err := c.RegisterTemp("scott/wang", "test")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)
	c.Close()

	wg.Wait()
}

func TestClientSuite(t *testing.T) {
	suite.Run(t, &ClientTestSuite{
		etcdConfig: struct {
			name      string
			endpoints []string
			timeout   time.Duration
			heartbeat int
		}{
			name:      "test",
			endpoints: []string{"localhost:2381"},
			timeout:   time.Second,
			heartbeat: 1,
		},
	})
}
