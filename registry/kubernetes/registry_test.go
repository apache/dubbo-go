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
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	v1 "k8s.io/api/core/v1"
)

var clientPodListJsonData = `{
    "apiVersion": "v1",
    "items": [
        {
            "apiVersion": "v1",
            "kind": "Pod",
            "metadata": {
                "annotations": {
                    "cni.projectcalico.org/containerID": "e7194ec69431b8f0d6eeedcc55dcb89360cce45840e6c448d57f0a2c4b5bb641",
                    "cni.projectcalico.org/podIP": "10.244.219.94/32",
                    "cni.projectcalico.org/podIPs": "10.244.219.94/32",
                    "dubbo.io/annotation": "W3siayI6Ii9kdWJiby9jb20uYXBhY2hlLmR1YmJvLnNhbXBsZS5iYXNpYy5JR3JlZXRlci9wcm92aWRlcnMiLCJ2IjoiIn0seyJrIjoiL2R1YmJvL2NvbS5hcGFjaGUuZHViYm8uc2FtcGxlLmJhc2ljLklHcmVldGVyL3Byb3ZpZGVycy9kdWJibyUzQSUyRiUyRjEwLjI0NC4yMTkuOTQlM0EyMDAwMCUyRmNvbS5hcGFjaGUuZHViYm8uc2FtcGxlLmJhc2ljLklHcmVldGVyJTNGYW55aG9zdCUzRHRydWUlMjZhcHAudmVyc2lvbiUzRDMuMC4wJTI2YXBwbGljYXRpb24lM0RkdWJiby5pbyUyNmJlYW4ubmFtZSUzREdyZWV0ZXJQcm92aWRlciUyNmNsdXN0ZXIlM0RmYWlsb3ZlciUyNmVudmlyb25tZW50JTNEZGV2JTI2ZXhwb3J0JTNEdHJ1ZSUyNmludGVyZmFjZSUzRGNvbS5hcGFjaGUuZHViYm8uc2FtcGxlLmJhc2ljLklHcmVldGVyJTI2aXAlM0QxMC4yNDQuMjE5Ljk0JTI2bG9hZGJhbGFuY2UlM0RyYW5kb20lMjZtZXNzYWdlX3NpemUlM0Q0JTI2bWV0YWRhdGEtdHlwZSUzRGxvY2FsJTI2bWV0aG9kcyUzRFNheUhlbGxvJTJDU2F5SGVsbG9TdHJlYW0lMjZtb2R1bGUlM0RzYW1wbGUlMjZuYW1lJTNEZHViYm8uaW8lMjZvcmdhbml6YXRpb24lM0RkdWJiby1nbyUyNm93bmVyJTNEZHViYm8tZ28lMjZwaWQlM0Q3JTI2cGlkJTNENyUyNnJlZ2lzdHJ5JTNEa3ViZXJuZXRlcyUyNnJlZ2lzdHJ5LnJvbGUlM0QzJTI2cmVsZWFzZSUzRGR1YmJvLWdvbGFuZy0zLjAuMCUyNnNlcnZpY2UuZmlsdGVyJTNEZWNobyUyQ21ldHJpY3MlMkN0b2tlbiUyQ2FjY2Vzc2xvZyUyQ3RwcyUyQ2dlbmVyaWNfc2VydmljZSUyQ2V4ZWN1dGUlMkNwc2h1dGRvd24lMjZzaWRlJTNEcHJvdmlkZXIlMjZ0aW1lc3RhbXAlM0QxNjQ3NDgzNjI1IiwidiI6IiJ9XQ=="
                },
                "creationTimestamp": "2022-03-17T02:20:12Z",
                "generateName": "dubbo-go-server-59bcfb86d4-",
                "labels": {
                    "dubbo.io/label": "dubbo.io.provider",
                    "pod-template-hash": "59bcfb86d4",
                    "role": "dubbo-go-server"
                },
                "name": "dubbo-go-server-59bcfb86d4-trs8t",
                "namespace": "default",
                "ownerReferences": [
                    {
                        "apiVersion": "apps/v1",
                        "blockOwnerDeletion": true,
                        "controller": true,
                        "kind": "ReplicaSet",
                        "name": "dubbo-go-server-59bcfb86d4",
                        "uid": "7491e556-66f7-4856-a54e-b30e23669efd"
                    }
                ],
                "resourceVersion": "7143545",
                "uid": "065ccf24-81a4-4964-b05a-bd3ebf52b76d"
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
                        "image": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-server:0.0.15",
                        "imagePullPolicy": "Always",
                        "name": "dubbo-go-server",
                        "resources": {},
                        "terminationMessagePath": "/dev/termination-log",
                        "terminationMessagePolicy": "File",
                        "volumeMounts": [
                            {
                                "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount",
                                "name": "kube-api-access-kvjwz",
                                "readOnly": true
                            }
                        ]
                    }
                ],
                "dnsPolicy": "ClusterFirst",
                "enableServiceLinks": true,
                "nodeName": "master",
                "preemptionPolicy": "PreemptLowerPriority",
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
                        "name": "kube-api-access-kvjwz",
                        "projected": {
                            "defaultMode": 420,
                            "sources": [
                                {
                                    "serviceAccountToken": {
                                        "expirationSeconds": 3607,
                                        "path": "token"
                                    }
                                },
                                {
                                    "configMap": {
                                        "items": [
                                            {
                                                "key": "ca.crt",
                                                "path": "ca.crt"
                                            }
                                        ],
                                        "name": "kube-root-ca.crt"
                                    }
                                },
                                {
                                    "downwardAPI": {
                                        "items": [
                                            {
                                                "fieldRef": {
                                                    "apiVersion": "v1",
                                                    "fieldPath": "metadata.namespace"
                                                },
                                                "path": "namespace"
                                            }
                                        ]
                                    }
                                }
                            ]
                        }
                    }
                ]
            },
            "status": {
                "conditions": [
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-17T02:20:12Z",
                        "status": "True",
                        "type": "Initialized"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-17T02:20:25Z",
                        "status": "True",
                        "type": "Ready"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-17T02:20:25Z",
                        "status": "True",
                        "type": "ContainersReady"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-17T02:20:12Z",
                        "status": "True",
                        "type": "PodScheduled"
                    }
                ],
                "containerStatuses": [
                    {
                        "containerID": "containerd://8096e139887be17d5f833b56a6be85aa3ca75715fb7773a2e240b9930d33fb4d",
                        "image": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-server:0.0.15",
                        "imageID": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-server@sha256:c3cb570a1cef8e7251181878a3a13d71758dfe544ab30dad1a10bf615b471247",
                        "lastState": {},
                        "name": "dubbo-go-server",
                        "ready": true,
                        "restartCount": 0,
                        "started": true,
                        "state": {
                            "running": {
                                "startedAt": "2022-03-17T02:20:25Z"
                            }
                        }
                    }
                ],
                "hostIP": "192.168.123.171",
                "phase": "Running",
                "podIP": "10.244.219.94",
                "podIPs": [
                    {
                        "ip": "10.244.219.94"
                    }
                ],
                "qosClass": "BestEffort",
                "startTime": "2022-03-17T02:20:12Z"
            }
        },
        {
            "apiVersion": "v1",
            "kind": "Pod",
            "metadata": {
                "annotations": {
                    "cni.projectcalico.org/containerID": "de0c50d1c53d3503054f24d57f1294adac79580f46c6a2936fe870a010d35276",
                    "cni.projectcalico.org/podIP": "10.244.104.11/32",
                    "cni.projectcalico.org/podIPs": "10.244.104.11/32",
                    "dubbo.io/annotation": "W3siayI6Ii9kdWJiby9jb20uYXBhY2hlLmR1YmJvLnNhbXBsZS5iYXNpYy5JR3JlZXRlci9wcm92aWRlcnMiLCJ2IjoiIn0seyJrIjoiL2R1YmJvL2NvbS5hcGFjaGUuZHViYm8uc2FtcGxlLmJhc2ljLklHcmVldGVyL3Byb3ZpZGVycy9kdWJibyUzQSUyRiUyRjEwLjI0NC4xMDQuMTElM0EyMDAwMCUyRmNvbS5hcGFjaGUuZHViYm8uc2FtcGxlLmJhc2ljLklHcmVldGVyJTNGYW55aG9zdCUzRHRydWUlMjZhcHAudmVyc2lvbiUzRDMuMC4wJTI2YXBwbGljYXRpb24lM0RkdWJiby5pbyUyNmJlYW4ubmFtZSUzREdyZWV0ZXJQcm92aWRlciUyNmNsdXN0ZXIlM0RmYWlsb3ZlciUyNmVudmlyb25tZW50JTNEZGV2JTI2ZXhwb3J0JTNEdHJ1ZSUyNmludGVyZmFjZSUzRGNvbS5hcGFjaGUuZHViYm8uc2FtcGxlLmJhc2ljLklHcmVldGVyJTI2aXAlM0QxMC4yNDQuMTA0LjExJTI2bG9hZGJhbGFuY2UlM0RyYW5kb20lMjZtZXNzYWdlX3NpemUlM0Q0JTI2bWV0YWRhdGEtdHlwZSUzRGxvY2FsJTI2bWV0aG9kcyUzRFNheUhlbGxvJTJDU2F5SGVsbG9TdHJlYW0lMjZtb2R1bGUlM0RzYW1wbGUlMjZuYW1lJTNEZHViYm8uaW8lMjZvcmdhbml6YXRpb24lM0RkdWJiby1nbyUyNm93bmVyJTNEZHViYm8tZ28lMjZwaWQlM0Q3JTI2cGlkJTNENyUyNnJlZ2lzdHJ5JTNEa3ViZXJuZXRlcyUyNnJlZ2lzdHJ5LnJvbGUlM0QzJTI2cmVsZWFzZSUzRGR1YmJvLWdvbGFuZy0zLjAuMCUyNnNlcnZpY2UuZmlsdGVyJTNEZWNobyUyQ21ldHJpY3MlMkN0b2tlbiUyQ2FjY2Vzc2xvZyUyQ3RwcyUyQ2dlbmVyaWNfc2VydmljZSUyQ2V4ZWN1dGUlMkNwc2h1dGRvd24lMjZzaWRlJTNEcHJvdmlkZXIlMjZ0aW1lc3RhbXAlM0QxNjQ3NDgzNjI0IiwidiI6IiJ9XQ=="
                },
                "creationTimestamp": "2022-03-17T02:20:12Z",
                "generateName": "dubbo-go-server-59bcfb86d4-",
                "labels": {
                    "dubbo.io/label": "dubbo.io.provider",
                    "pod-template-hash": "59bcfb86d4",
                    "role": "dubbo-go-server"
                },
                "name": "dubbo-go-server-59bcfb86d4-vrj2v",
                "namespace": "default",
                "ownerReferences": [
                    {
                        "apiVersion": "apps/v1",
                        "blockOwnerDeletion": true,
                        "controller": true,
                        "kind": "ReplicaSet",
                        "name": "dubbo-go-server-59bcfb86d4",
                        "uid": "7491e556-66f7-4856-a54e-b30e23669efd"
                    }
                ],
                "resourceVersion": "7143615",
                "uid": "560fdd87-4b77-4c37-840f-f05d5be7f726"
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
                        "image": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-server:0.0.15",
                        "imagePullPolicy": "Always",
                        "name": "dubbo-go-server",
                        "resources": {},
                        "terminationMessagePath": "/dev/termination-log",
                        "terminationMessagePolicy": "File",
                        "volumeMounts": [
                            {
                                "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount",
                                "name": "kube-api-access-mq7sw",
                                "readOnly": true
                            }
                        ]
                    }
                ],
                "dnsPolicy": "ClusterFirst",
                "enableServiceLinks": true,
                "nodeName": "node2",
                "preemptionPolicy": "PreemptLowerPriority",
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
                        "name": "kube-api-access-mq7sw",
                        "projected": {
                            "defaultMode": 420,
                            "sources": [
                                {
                                    "serviceAccountToken": {
                                        "expirationSeconds": 3607,
                                        "path": "token"
                                    }
                                },
                                {
                                    "configMap": {
                                        "items": [
                                            {
                                                "key": "ca.crt",
                                                "path": "ca.crt"
                                            }
                                        ],
                                        "name": "kube-root-ca.crt"
                                    }
                                },
                                {
                                    "downwardAPI": {
                                        "items": [
                                            {
                                                "fieldRef": {
                                                    "apiVersion": "v1",
                                                    "fieldPath": "metadata.namespace"
                                                },
                                                "path": "namespace"
                                            }
                                        ]
                                    }
                                }
                            ]
                        }
                    }
                ]
            },
            "status": {
                "conditions": [
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-17T02:19:49Z",
                        "status": "True",
                        "type": "Initialized"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-17T02:20:24Z",
                        "status": "True",
                        "type": "Ready"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-17T02:20:24Z",
                        "status": "True",
                        "type": "ContainersReady"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-17T02:20:12Z",
                        "status": "True",
                        "type": "PodScheduled"
                    }
                ],
                "containerStatuses": [
                    {
                        "containerID": "containerd://e654db7becd53411f21f8ef9b86e5e666d74b3554d7efe577f000f347240b628",
                        "image": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-server:0.0.15",
                        "imageID": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-server@sha256:c3cb570a1cef8e7251181878a3a13d71758dfe544ab30dad1a10bf615b471247",
                        "lastState": {},
                        "name": "dubbo-go-server",
                        "ready": true,
                        "restartCount": 0,
                        "started": true,
                        "state": {
                            "running": {
                                "startedAt": "2022-03-17T02:20:24Z"
                            }
                        }
                    }
                ],
                "hostIP": "192.168.123.180",
                "phase": "Running",
                "podIP": "10.244.104.11",
                "podIPs": [
                    {
                        "ip": "10.244.104.11"
                    }
                ],
                "qosClass": "BestEffort",
                "startTime": "2022-03-17T02:19:49Z"
            }
        }
    ],
    "kind": "List",
    "metadata": {
        "resourceVersion": "",
        "selfLink": ""
    }
}`

func getTestRegistry(t *testing.T) *kubernetesRegistry {

	const (
		podNameKey              = "HOSTNAME"
		nameSpaceKey            = "NAMESPACE"
		needWatchedNameSpaceKey = "DUBBO_NAMESPACE"
	)
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

	regurl, err := common.NewURL("registry://127.0.0.1:443", common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER)))
	if err != nil {
		t.Fatal(err)
	}
	out, err := newMockKubernetesRegistry(regurl, pl)
	if err != nil {
		t.Fatal(err)
	}

	return out.(*kubernetesRegistry)
}

func TestRegister(t *testing.T) {

	r := getTestRegistry(t)
	defer r.Destroy()

	url, _ := common.NewURL(
		"dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider",
		common.WithParamsValue(constant.ClusterKey, "mock"),
		common.WithMethods([]string{"GetUser", "AddUser"}),
	)

	err := r.Register(url)
	assert.NoError(t, err)
	_, _, err = r.client.GetChildren("/dubbo/com.ikurento.user.UserProvider/providers")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSubscribe(t *testing.T) {

	r := getTestRegistry(t)
	defer r.Destroy()

	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.ClusterKey, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))
	if err != nil {
		t.Fatal(err)
	}

	listener, err := r.DoSubscribe(url)
	if err != nil {
		t.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {

		defer wg.Done()
		registerErr := r.Register(url)
		if registerErr != nil {
			t.Error(registerErr)
		}
	}()

	wg.Wait()

	serviceEvent, err := listener.Next()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("get service event %s", serviceEvent)
}

func TestConsumerDestroy(t *testing.T) {

	r := getTestRegistry(t)

	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider",
		common.WithParamsValue(constant.ClusterKey, "mock"),
		common.WithMethods([]string{"GetUser", "AddUser"}))

	_, err := r.DoSubscribe(url)
	if err != nil {
		t.Fatal(err)
	}

	//listener.Close()
	time.Sleep(1e9)
	r.Destroy()

	assert.Equal(t, false, r.IsAvailable())

}

func TestProviderDestroy(t *testing.T) {

	r := getTestRegistry(t)

	url, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider",
		common.WithParamsValue(constant.ClusterKey, "mock"),
		common.WithMethods([]string{"GetUser", "AddUser"}))
	err := r.Register(url)
	assert.NoError(t, err)

	time.Sleep(1e9)
	r.Destroy()
	assert.Equal(t, false, r.IsAvailable())
}

func TestNewRegistry(t *testing.T) {

	regUrl, err := common.NewURL("registry://127.0.0.1:443",
		common.WithParamsValue(constant.RegistryRoleKey, strconv.Itoa(common.PROVIDER)))
	if err != nil {
		t.Fatal(err)
	}
	_, err = newKubernetesRegistry(regUrl)
	if err == nil {
		t.Fatal("not in cluster, should be a err")
	}
}

func TestHandleClientRestart(t *testing.T) {

	r := getTestRegistry(t)
	r.WaitGroup().Add(1)
	go r.HandleClientRestart()
	time.Sleep(timeSecondDuration(1))
	r.client.Close()
}
