package etcdv3

import (
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/juju/errors"
	"google.golang.org/grpc/connectivity"
)

// etcd connect config
var (
	name      = "test"
	timeout   = time.Second
	heartbeat = 1
	endpoints = []string{"localhost:2379"}
)

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

func initClient(t *testing.T) *Client {

	c, err := newClient(name, endpoints, timeout, heartbeat)
	if err != nil {
		t.Fatal(err)
	}
	c.CleanKV()
	return c
}

func TestMain(m *testing.M) {

	startETCDServer()
	m.Run()
	stopETCDServer()
}

func startETCDServer() {

	cmd := exec.Command("./load.sh", "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	cmd.Dir = "./single"

	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func stopETCDServer() {
	cmd := exec.Command("./load.sh", "stop")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	cmd.Dir = "./single"

	if err := cmd.Run(); err != nil {
		panic(err)
	}

}

func Test_newClient(t *testing.T) {

	c := initClient(t)
	defer c.Close()

	if c.rawClient.ActiveConnection().GetState() != connectivity.Ready {
		t.Fatal(c.rawClient.ActiveConnection().GetState())
	}
}

func TestClient_Close(t *testing.T) {

	c := initClient(t)
	c.Close()
}

func TestClient_Create(t *testing.T) {

	tests := tests

	c := initClient(t)
	defer c.Close()

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

func TestClient_Delete(t *testing.T) {

	tests := tests

	c := initClient(t)
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
		if errors.Cause(err) == expect {
			continue
		}

		if err != nil {
			t.Fatal(err)
		}
	}

}

func TestClient_GetChildrenKVList(t *testing.T) {

	tests := tests

	c := initClient(t)
	defer c.Close()

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

func TestClient_Watch(t *testing.T) {

	tests := tests

	c := initClient(t)
	defer c.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {

		defer wg.Done()

		wc, err := c.watch(prefix)
		if err != nil {
			t.Fatal(err)
		}

		for e := range wc {

			if e.Err() != nil {
				t.Fatal(err)
			}

			for _, event := range e.Events {
				t.Logf("type IsCreate %v k %s v %s", event.IsCreate(), event.Kv.Key, event.Kv.Value)
			}
		}

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

func TestClient_RegisterTemp(t *testing.T) {

	c := initClient(t)
	observeC := initClient(t)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		completePath := path.Join("scott", "wang")
		wc, err := observeC.watch(completePath)
		if err != nil {
			t.Fatal(err)
		}

		for e := range wc {

			for _, event := range e.Events {

				if e.Err() != nil {
					t.Fatal(e.Err())
				}
				if event.Type == mvccpb.DELETE {
					t.Logf("complete key (%s) is delete", completePath)
					wg.Done()
					observeC.Close()
					return
				}
				t.Logf("type IsCreate %v k %s v %s", event.IsCreate(), event.Kv.Key, event.Kv.Value)
			}
		}
	}()

	_, err := c.RegisterTemp("scott", "wang")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)
	c.Close()

	wg.Wait()
}

func TestClient_Valid(t *testing.T) {

	c := initClient(t)

	if c.Valid() != true {
		t.Fatal("client is not valid")
	}

	c.Close()

	if c.Valid() != false {

		t.Fatal("client is valid")

	}

}

func TestClient_Done(t *testing.T) {

	c := initClient(t)

	go func() {
		time.Sleep(2 * time.Second)
		c.Close()
	}()

	c.Wait.Wait()
}
