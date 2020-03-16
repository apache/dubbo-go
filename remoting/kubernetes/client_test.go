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
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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

var clientPodJsonData = `{
    "apiVersion": "v1",
    "kind": "Pod",
    "metadata": {
        "annotations": {
            "dubbo.io/annotation": "W3siayI6Ii9kdWJibyIsInYiOiIifSx7ImsiOiIvZHViYm8vY29tLmlrdXJlbnRvLnVzZXIuVXNlclByb3ZpZGVyIiwidiI6IiJ9LHsiayI6Ii9kdWJiby9jb20uaWt1cmVudG8udXNlci5Vc2VyUHJvdmlkZXIvY29uc3VtZXJzIiwidiI6IiJ9LHsiayI6Ii9kdWJibyIsInYiOiIifSx7ImsiOiIvZHViYm8vY29tLmlrdXJlbnRvLnVzZXIuVXNlclByb3ZpZGVyIiwidiI6IiJ9LHsiayI6Ii9kdWJiby9jb20uaWt1cmVudG8udXNlci5Vc2VyUHJvdmlkZXIvcHJvdmlkZXJzIiwidiI6IiJ9LHsiayI6Ii9kdWJiby9jb20uaWt1cmVudG8udXNlci5Vc2VyUHJvdmlkZXIvY29uc3VtZXJzL2NvbnN1bWVyJTNBJTJGJTJGMTcyLjE3LjAuOCUyRlVzZXJQcm92aWRlciUzRmNhdGVnb3J5JTNEY29uc3VtZXJzJTI2ZHViYm8lM0RkdWJib2dvLWNvbnN1bWVyLTIuNi4wJTI2cHJvdG9jb2wlM0RkdWJibyIsInYiOiIifV0="
        },
        "creationTimestamp": "2020-03-13T03:38:57Z",
        "labels": {
            "dubbo.io/label": "dubbo.io-value"
        },
        "name": "client",
        "namespace": "default",
        "resourceVersion": "2449700",
        "selfLink": "/api/v1/namespaces/default/pods/client",
        "uid": "3ec394f5-dcc6-49c3-8061-57b4b2b41344"
    },
    "spec": {
        "containers": [
            {
                "env": [
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
                "image": "registry.cn-hangzhou.aliyuncs.com/scottwang/dubbogo-client",
                "imagePullPolicy": "Always",
                "name": "client",
                "resources": {},
                "terminationMessagePath": "/dev/termination-log",
                "terminationMessagePolicy": "File",
                "volumeMounts": [
                    {
                        "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount",
                        "name": "dubbo-sa-token-l2lzh",
                        "readOnly": true
                    }
                ]
            }
        ],
        "dnsPolicy": "ClusterFirst",
        "enableServiceLinks": true,
        "nodeName": "minikube",
        "priority": 0,
        "restartPolicy": "Never",
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
                "name": "dubbo-sa-token-l2lzh",
                "secret": {
                    "defaultMode": 420,
                    "secretName": "dubbo-sa-token-l2lzh"
                }
            }
        ]
    },
    "status": {
        "conditions": [
            {
                "lastProbeTime": null,
                "lastTransitionTime": "2020-03-13T03:38:57Z",
                "status": "True",
                "type": "Initialized"
            },
            {
                "lastProbeTime": null,
                "lastTransitionTime": "2020-03-13T03:40:18Z",
                "status": "True",
                "type": "Ready"
            },
            {
                "lastProbeTime": null,
                "lastTransitionTime": "2020-03-13T03:40:18Z",
                "status": "True",
                "type": "ContainersReady"
            },
            {
                "lastProbeTime": null,
                "lastTransitionTime": "2020-03-13T03:38:57Z",
                "status": "True",
                "type": "PodScheduled"
            }
        ],
        "containerStatuses": [
            {
                "containerID": "docker://2870d6abc19ca7fe22ca635ebcfac5d48c6d5550a659bafd74fb48104f6dfe3c",
                "image": "registry.cn-hangzhou.aliyuncs.com/scottwang/dubbogo-client:latest",
                "imageID": "docker-pullable://registry.cn-hangzhou.aliyuncs.com/scottwang/dubbogo-client@sha256:1f075131f708a0d400339e81549d7c4d4ed917ab0b6bd38ef458dd06ad25a559",
                "lastState": {},
                "name": "client",
                "ready": true,
                "restartCount": 0,
                "state": {
                    "running": {
                        "startedAt": "2020-03-13T03:40:17Z"
                    }
                }
            }
        ],
        "hostIP": "10.0.2.15",
        "phase": "Running",
        "podIP": "172.17.0.8",
        "qosClass": "BestEffort",
        "startTime": "2020-03-13T03:38:57Z"
    }
}
`

type KubernetesClientTestSuite struct {
	suite.Suite

	currentPod v1.Pod
}

func (s *KubernetesClientTestSuite) initClient() *Client {

	t := s.T()

	client, err := newMockClient(s.currentPod.GetNamespace(), func() (kubernetes.Interface, error) {

		out := fake.NewSimpleClientset()

		// mock current pod
		if _, err := out.CoreV1().Pods(s.currentPod.GetNamespace()).Create(&s.currentPod); err != nil {
			t.Fatal(err)
		}
		return out, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func (s *KubernetesClientTestSuite) SetupSuite() {

	t := s.T()

	// 1. install test data
	if err := json.Unmarshal([]byte(clientPodJsonData), &s.currentPod); err != nil {
		t.Fatal(err)
	}

	// 2. set downward-api inject env
	if err := os.Setenv(podNameKey, s.currentPod.GetName()); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv(nameSpaceKey, s.currentPod.GetNamespace()); err != nil {
		t.Fatal(err)
	}

}

func (s *KubernetesClientTestSuite) TestReadCurrentPodName() {
	t := s.T()

	n, err := getCurrentPodName()
	if err != nil {
		t.Fatal(err)
	}

	if n != s.currentPod.GetName() {
		t.Fatalf("expect %s but got %s", s.currentPod.GetName(), n)
	}

}
func (s *KubernetesClientTestSuite) TestReadCurrentNameSpace() {
	t := s.T()

	ns, err := getCurrentNameSpace()
	if err != nil {
		t.Fatal(err)
	}

	if ns != s.currentPod.GetNamespace() {
		t.Fatalf("expect %s but got %s", s.currentPod.GetNamespace(), ns)
	}

}
func (s *KubernetesClientTestSuite) TestClientValid() {

	t := s.T()

	client := s.initClient()
	defer client.Close()

	if client.Valid() != true {
		t.Fatal("client is not valid")
	}

	client.Close()
	if client.Valid() != false {
		t.Fatal("client is valid")
	}
}

func (s *KubernetesClientTestSuite) TestClientDone() {

	t := s.T()

	client := s.initClient()

	go func() {
		time.Sleep(time.Second)
		client.Close()
	}()

	<-client.Done()

	if client.Valid() == true {
		t.Fatal("client should be invalid then")
	}
}

func (s *KubernetesClientTestSuite) TestClientCreateKV() {

	t := s.T()

	client := s.initClient()
	defer client.Close()

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v

		if err := client.Create(k, v); err != nil {
			t.Fatal(err)
		}

	}
}

func (s *KubernetesClientTestSuite) TestClientGetChildrenKVList() {

	t := s.T()

	client := s.initClient()
	defer client.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	syncDataComplete := make(chan struct{})

	go func() {

		wc, done, err := client.WatchWithPrefix(prefix)
		if err != nil {
			t.Fatal(err)
		}
		i := 0
		wg.Done()

		for {
			select {
			case e := <-wc:
				i++
				t.Logf("got event %v k %s v %s", e.EventType, e.Key, e.Value)
				if i == 3 {
					// already sync all event
					syncDataComplete <- struct{}{}
					return
				}
			case <-done:
				t.Log("the watcherSet watcher was stopped")
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
		t.Fatal(err)
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

func (s *KubernetesClientTestSuite) TestClientWatchPrefix() {

	t := s.T()

	client := s.initClient()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {

		wc, done, err := client.WatchWithPrefix(prefix)
		if err != nil {
			t.Fatal(err)
		}

		wg.Done()

		for {
			select {
			case e := <-wc:
				t.Logf("got event %v k %s v %s", e.EventType, e.Key, e.Value)
			case <-done:
				t.Log("the watcherSet watcher was stopped")
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

func (s *KubernetesClientTestSuite) TestNewClient() {

	t := s.T()

	_, err := newClient(s.currentPod.GetNamespace())
	if err == nil {
		t.Fatal("the out of cluster test should fail")
	}

}

func (s *KubernetesClientTestSuite) TestClientWatch() {

	t := s.T()

	client := s.initClient()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {

		wc, done, err := client.Watch(prefix)
		if err != nil {
			t.Fatal(err)
		}
		wg.Done()

		for {
			select {
			case e := <-wc:
				t.Logf("got event %v k %s v %s", e.EventType, e.Key, e.Value)
			case <-done:
				t.Log("the watcherSet watcher was stopped")
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

func TestKubernetesClient(t *testing.T) {
	suite.Run(t, new(KubernetesClientTestSuite))
}
