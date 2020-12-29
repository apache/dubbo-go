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
	"encoding/json"
	"fmt"
	_ "net/http/pprof"
	"os"
	"strings"
	"sync"
	"testing"
)

import (
	v1 "k8s.io/api/core/v1"
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

var (
	watcherStopLog = "the watcherSet watcher was stopped"
)
var clientPodListJsonData = `{
    "apiVersion": "v1",
    "items": [
        {
            "apiVersion": "v1",
            "kind": "Pod",
            "metadata": {
                "annotations": {
                    "dubbo.io/annotation": "W3siayI6Ii9kdWJiby9jb20uaWt1cmVudG8udXNlci5Vc2VyUHJvdmlkZXIvcHJvdmlkZXJzIiwidiI6IiJ9LHsiayI6Ii9kdWJiby9jb20uaWt1cmVudG8udXNlci5Vc2VyUHJvdmlkZXIvcHJvdmlkZXJzL2R1YmJvJTNBJTJGJTJGMTcyLjE3LjAuNiUzQTIwMDAwJTJGVXNlclByb3ZpZGVyJTNGYWNjZXNzbG9nJTNEJTI2YW55aG9zdCUzRHRydWUlMjZhcHAudmVyc2lvbiUzRDAuMC4xJTI2YXBwbGljYXRpb24lM0RCRFRTZXJ2aWNlJTI2YXV0aCUzRCUyNmJlYW4ubmFtZSUzRFVzZXJQcm92aWRlciUyNmNsdXN0ZXIlM0RmYWlsb3ZlciUyNmVudmlyb25tZW50JTNEZGV2JTI2ZXhlY3V0ZS5saW1pdCUzRCUyNmV4ZWN1dGUubGltaXQucmVqZWN0ZWQuaGFuZGxlciUzRCUyNmdyb3VwJTNEJTI2aW50ZXJmYWNlJTNEY29tLmlrdXJlbnRvLnVzZXIuVXNlclByb3ZpZGVyJTI2aXAlM0QxNzIuMTcuMC42JTI2bG9hZGJhbGFuY2UlM0RyYW5kb20lMjZtZXRob2RzLkdldFVzZXIubG9hZGJhbGFuY2UlM0RyYW5kb20lMjZtZXRob2RzLkdldFVzZXIucmV0cmllcyUzRDElMjZtZXRob2RzLkdldFVzZXIudHBzLmxpbWl0LmludGVydmFsJTNEJTI2bWV0aG9kcy5HZXRVc2VyLnRwcy5saW1pdC5yYXRlJTNEJTI2bWV0aG9kcy5HZXRVc2VyLnRwcy5saW1pdC5zdHJhdGVneSUzRCUyNm1ldGhvZHMuR2V0VXNlci53ZWlnaHQlM0QwJTI2bW9kdWxlJTNEZHViYm9nbyUyQnVzZXItaW5mbyUyQnNlcnZlciUyNm5hbWUlM0RCRFRTZXJ2aWNlJTI2b3JnYW5pemF0aW9uJTNEaWt1cmVudG8uY29tJTI2b3duZXIlM0RaWCUyNnBhcmFtLnNpZ24lM0QlMjZwaWQlM0Q2JTI2cmVnaXN0cnkucm9sZSUzRDMlMjZyZWxlYXNlJTNEZHViYm8tZ29sYW5nLTEuMy4wJTI2cmV0cmllcyUzRCUyNnNlcnZpY2UuZmlsdGVyJTNEZWNobyUyNTJDdG9rZW4lMjUyQ2FjY2Vzc2xvZyUyNTJDdHBzJTI1MkNnZW5lcmljX3NlcnZpY2UlMjUyQ2V4ZWN1dGUlMjUyQ3BzaHV0ZG93biUyNnNpZGUlM0Rwcm92aWRlciUyNnRpbWVzdGFtcCUzRDE1OTExNTYxNTUlMjZ0cHMubGltaXQuaW50ZXJ2YWwlM0QlMjZ0cHMubGltaXQucmF0ZSUzRCUyNnRwcy5saW1pdC5yZWplY3RlZC5oYW5kbGVyJTNEJTI2dHBzLmxpbWl0LnN0cmF0ZWd5JTNEJTI2dHBzLmxpbWl0ZXIlM0QlMjZ2ZXJzaW9uJTNEJTI2d2FybXVwJTNEMTAwIiwidiI6IiJ9XQ=="
                },
                "creationTimestamp": "2020-06-03T03:49:14Z",
                "generateName": "server-84c864f5bc-",
                "labels": {
                    "dubbo.io/label": "dubbo.io-value",
                    "pod-template-hash": "84c864f5bc",
                    "role": "server"
                },
                "name": "server-84c864f5bc-r8qvz",
                "namespace": "default",
                "ownerReferences": [
                    {
                        "apiVersion": "apps/v1",
                        "blockOwnerDeletion": true,
                        "controller": true,
                        "kind": "ReplicaSet",
                        "name": "server-84c864f5bc",
                        "uid": "fa376dbb-4f37-4705-8e80-727f592c19b3"
                    }
                ],
                "resourceVersion": "517460",
                "selfLink": "/api/v1/namespaces/default/pods/server-84c864f5bc-r8qvz",
                "uid": "f4fc811c-200c-4445-8d4f-532144957dcc"
            },
            "spec": {
                "containers": [
                    {
                        "env": [
                            {
                                "name": "DUBBO_NAMESPACE",
                                "value": "default"
                            },
                            {
                                "name": "NAMESPACE",
                                "valueFrom": {
                                    "fieldRef": {
                                        "apiVersion": "v1",
                                        "fieldPath": "metadata.namespace"
                                    }
                                }
                            }
                        ],
                        "image": "192.168.240.101:5000/scott/go-server",
                        "imagePullPolicy": "Always",
                        "name": "server",
                        "resources": {},
                        "terminationMessagePath": "/dev/termination-log",
                        "terminationMessagePolicy": "File",
                        "volumeMounts": [
                            {
                                "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount",
                                "name": "dubbo-sa-token-5qbtb",
                                "readOnly": true
                            }
                        ]
                    }
                ],
                "dnsPolicy": "ClusterFirst",
                "enableServiceLinks": true,
                "nodeName": "minikube",
                "priority": 0,
                "restartPolicy": "Always",
                "schedulerName": "default-scheduler",
                "securityContext": {},
                "serviceAccount": "dubbo-sa",
                "serviceAccountName": "dubbo-sa",
                "terminationGracePeriodSeconds": 30,
                "tolerations": [
                    {
                        "effect": "NoExecute",
                        "key": "node.kubernetes.io/not-ready",
                        "operator": "Exists",
                        "tolerationSeconds": 300
                    },
                    {
                        "effect": "NoExecute",
                        "key": "node.kubernetes.io/unreachable",
                        "operator": "Exists",
                        "tolerationSeconds": 300
                    }
                ],
                "volumes": [
                    {
                        "name": "dubbo-sa-token-5qbtb",
                        "secret": {
                            "defaultMode": 420,
                            "secretName": "dubbo-sa-token-5qbtb"
                        }
                    }
                ]
            },
            "status": {
                "conditions": [
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2020-06-03T03:49:14Z",
                        "status": "True",
                        "type": "Initialized"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2020-06-03T03:49:15Z",
                        "status": "True",
                        "type": "Ready"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2020-06-03T03:49:15Z",
                        "status": "True",
                        "type": "ContainersReady"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2020-06-03T03:49:14Z",
                        "status": "True",
                        "type": "PodScheduled"
                    }
                ],
                "containerStatuses": [
                    {
                        "containerID": "docker://b6421e05ce44f8a1c4fa6b72274980777c7c0f945516209f7c0558cd0cd65406",
                        "image": "192.168.240.101:5000/scott/go-server:latest",
                        "imageID": "docker-pullable://192.168.240.101:5000/scott/go-server@sha256:4eecf895054f0ff93d80db64992a561d10504e55582def6dcb6093a6d6d92461",
                        "lastState": {},
                        "name": "server",
                        "ready": true,
                        "restartCount": 0,
                        "started": true,
                        "state": {
                            "running": {
                                "startedAt": "2020-06-03T03:49:15Z"
                            }
                        }
                    }
                ],
                "hostIP": "10.0.2.15",
                "phase": "Running",
                "podIP": "172.17.0.6",
                "podIPs": [
                    {
                        "ip": "172.17.0.6"
                    }
                ],
                "qosClass": "BestEffort",
                "startTime": "2020-06-03T03:49:14Z"
            }
        }
    ],
    "kind": "List",
    "metadata": {
        "resourceVersion": "",
        "selfLink": ""
    }
}
`

func getTestClient(t *testing.T) *Client {

	pl := &v1.PodList{}
	// 1. install test data
	if err := json.Unmarshal([]byte(clientPodListJsonData), &pl); err != nil {
		t.Fatal(err)
	}
	currentPod := pl.Items[0]

	env := map[string]string{
		nameSpaceKey:            currentPod.GetNamespace(),
		podNameKey:              currentPod.GetName(),
		needWatchedNameSpaceKey: "default",
	}

	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			t.Fatal(err)
		}
	}

	client, err := NewMockClient(pl)
	if err != nil {
		t.Fatal(err)
	}

	return client
}

func TestClientValid(t *testing.T) {

	client := getTestClient(t)
	defer client.Close()

	if !client.Valid() {
		t.Fatal("client is not valid")
	}

	client.Close()
	if client.Valid() {
		t.Fatal("client is valid")
	}
}

func TestClientDone(t *testing.T) {

	client := getTestClient(t)

	go func() {
		client.Close()
	}()

	<-client.Done()

	if client.Valid() {
		t.Fatal("client should be invalid")
	}
}

func TestClientCreateKV(t *testing.T) {

	client := getTestClient(t)
	defer client.Close()

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v

		if err := client.Create(k, v); err != nil {
			t.Fatal(err)
		}

	}
}

func TestClientGetChildrenKVList(t *testing.T) {

	client := getTestClient(t)
	defer client.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	syncDataComplete := make(chan struct{})

	go func() {

		wc, done, err := client.WatchWithPrefix(prefix)
		if err != nil {
			t.Fatal(err)
		}

		wg.Done()
		i := 0

		for {
			select {
			case e := <-wc:
				i++
				fmt.Printf("got event %v k %s v %s\n", e.EventType, e.Key, e.Value)
				if i == 3 {
					// already sync all event
					syncDataComplete <- struct{}{}
					return
				}
			case <-done:
				t.Log(watcherStopLog)
				return
			}
		}
	}()

	// wait the watch goroutine start
	wg.Wait()

	expect := make(map[string]string)
	got := make(map[string]string)

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v

		if strings.Contains(k, prefix) {
			expect[k] = v
		}

		if err := client.Create(k, v); err != nil {
			t.Fatal(err)
		}
	}

	<-syncDataComplete

	// start get all children
	kList, vList, err := client.GetChildren(prefix)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < len(kList); i++ {
		got[kList[i]] = vList[i]
	}

	for expectK, expectV := range expect {

		if got[expectK] != expectV {
			t.Fatalf("expect {%s: %s} but got {%s: %v}", expectK, expectV, expectK, got[expectK])
		}
	}

}

func TestClientWatchPrefix(t *testing.T) {

	client := getTestClient(t)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {

		wc, done, err := client.WatchWithPrefix(prefix)
		if err != nil {
			t.Error(err)
		}

		wg.Done()

		for {
			select {
			case e := <-wc:
				t.Logf("got event %v k %s v %s", e.EventType, e.Key, e.Value)
			case <-done:
				t.Log(watcherStopLog)
				return
			}
		}
	}()

	// must wait the watch goroutine work
	wg.Wait()

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v

		if err := client.Create(k, v); err != nil {
			t.Fatal(err)
		}
	}

	client.Close()
}

func TestClientWatch(t *testing.T) {

	client := getTestClient(t)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {

		wc, done, err := client.Watch(prefix)
		if err != nil {
			t.Error(err)
		}
		wg.Done()

		for {
			select {
			case e := <-wc:
				t.Logf("got event %v k %s v %s", e.EventType, e.Key, e.Value)
			case <-done:
				t.Log(watcherStopLog)
				return
			}
		}

	}()

	// must wait the watch goroutine already start the watch goroutine
	wg.Wait()

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v

		if err := client.Create(k, v); err != nil {
			t.Fatal(err)
		}
	}

	client.Close()
}
