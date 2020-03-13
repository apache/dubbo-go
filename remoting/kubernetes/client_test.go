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
	"github.com/pkg/errors"
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

var server1PodJsonData = `{
    "apiVersion": "v1",
    "kind": "Pod",
    "metadata": {
        "annotations": {
            "dubbo.io/annotation": "W3siayI6Ii9kdWJibyIsInYiOiIifSx7ImsiOiIvZHViYm8vY29tLmlrdXJlbnRvLnVzZXIuVXNlclByb3ZpZGVyIiwidiI6IiJ9LHsiayI6Ii9kdWJiby9jb20uaWt1cmVudG8udXNlci5Vc2VyUHJvdmlkZXIvcHJvdmlkZXJzIiwidiI6IiJ9LHsiayI6Ii9kdWJiby9jb20uaWt1cmVudG8udXNlci5Vc2VyUHJvdmlkZXIvcHJvdmlkZXJzL2R1YmJvJTNBJTJGJTJGMTcyLjE3LjAuNyUzQTIwMDAwJTJGVXNlclByb3ZpZGVyJTNGYWNjZXNzbG9nJTNEJTI2YW55aG9zdCUzRHRydWUlMjZhcHAudmVyc2lvbiUzRDAuMC4xJTI2YXBwbGljYXRpb24lM0RCRFRTZXJ2aWNlJTI2YmVhbi5uYW1lJTNEVXNlclByb3ZpZGVyJTI2Y2F0ZWdvcnklM0Rwcm92aWRlcnMlMjZjbHVzdGVyJTNEZmFpbG92ZXIlMjZkdWJibyUzRGR1YmJvLXByb3ZpZGVyLWdvbGFuZy0yLjYuMCUyNmVudmlyb25tZW50JTNEZGV2JTI2ZXhlY3V0ZS5saW1pdCUzRCUyNmV4ZWN1dGUubGltaXQucmVqZWN0ZWQuaGFuZGxlciUzRCUyNmdyb3VwJTNEJTI2aW50ZXJmYWNlJTNEY29tLmlrdXJlbnRvLnVzZXIuVXNlclByb3ZpZGVyJTI2aXAlM0QxNzIuMTcuMC43JTI2bG9hZGJhbGFuY2UlM0RyYW5kb20lMjZtZXRob2RzLkdldFVzZXIubG9hZGJhbGFuY2UlM0RyYW5kb20lMjZtZXRob2RzLkdldFVzZXIucmV0cmllcyUzRDElMjZtZXRob2RzLkdldFVzZXIudHBzLmxpbWl0LmludGVydmFsJTNEJTI2bWV0aG9kcy5HZXRVc2VyLnRwcy5saW1pdC5yYXRlJTNEJTI2bWV0aG9kcy5HZXRVc2VyLnRwcy5saW1pdC5zdHJhdGVneSUzRCUyNm1ldGhvZHMuR2V0VXNlci53ZWlnaHQlM0QwJTI2bW9kdWxlJTNEZHViYm9nbyUyQnVzZXItaW5mbyUyQnNlcnZlciUyNm5hbWUlM0RCRFRTZXJ2aWNlJTI2b3JnYW5pemF0aW9uJTNEaWt1cmVudG8uY29tJTI2b3duZXIlM0RaWCUyNnBpZCUzRDEwJTI2cmVnaXN0cnkucm9sZSUzRDMlMjZyZXRyaWVzJTNEJTI2c2VydmljZS5maWx0ZXIlM0RlY2hvJTI1MkN0b2tlbiUyNTJDYWNjZXNzbG9nJTI1MkN0cHMlMjUyQ2V4ZWN1dGUlMjZzaWRlJTNEcHJvdmlkZXIlMjZ0aW1lc3RhbXAlM0QxNTg0MDcwODEwJTI2dHBzLmxpbWl0LmludGVydmFsJTNEJTI2dHBzLmxpbWl0LnJhdGUlM0QlMjZ0cHMubGltaXQucmVqZWN0ZWQuaGFuZGxlciUzRCUyNnRwcy5saW1pdC5zdHJhdGVneSUzRCUyNnRwcy5saW1pdGVyJTNEJTI2dmVyc2lvbiUzRCUyNndhcm11cCUzRDEwMCIsInYiOiIifV0="
        },
        "creationTimestamp": "2020-03-13T03:38:57Z",
        "generateName": "server-5b8f9f85c6-",
        "labels": {
            "dubbo.io/label": "dubbo.io-value",
            "pod-template-hash": "5b8f9f85c6",
            "role": "server"
        },
        "name": "server-5b8f9f85c6-2w5rq",
        "namespace": "default",
        "ownerReferences": [
            {
                "apiVersion": "apps/v1",
                "blockOwnerDeletion": true,
                "controller": true,
                "kind": "ReplicaSet",
                "name": "server-5b8f9f85c6",
                "uid": "65e9d2b0-f286-4b21-ac31-260f1412556b"
            }
        ],
        "resourceVersion": "2449678",
        "selfLink": "/api/v1/namespaces/default/pods/server-5b8f9f85c6-2w5rq",
        "uid": "ae7497c7-396d-40c5-b53e-5720a696e4ee"
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
                "image": "registry.cn-hangzhou.aliyuncs.com/scottwang/dubbogo-server",
                "imagePullPolicy": "Always",
                "name": "server",
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
                "lastTransitionTime": "2020-03-13T03:40:10Z",
                "status": "True",
                "type": "Ready"
            },
            {
                "lastProbeTime": null,
                "lastTransitionTime": "2020-03-13T03:40:10Z",
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
                "containerID": "docker://88144bc6eabf783c0954c2e078a7270c8a0246d6f8af081dfcc0956e8c3cd2de",
                "image": "registry.cn-hangzhou.aliyuncs.com/scottwang/dubbogo-server:latest",
                "imageID": "docker-pullable://registry.cn-hangzhou.aliyuncs.com/scottwang/dubbogo-server@sha256:60654ddba3a16ca3de52c8e30650a4c1d8b2ed8f8542af489b7a5a459e46fe6b",
                "lastState": {},
                "name": "server",
                "ready": true,
                "restartCount": 0,
                "state": {
                    "running": {
                        "startedAt": "2020-03-13T03:40:10Z"
                    }
                }
            }
        ],
        "hostIP": "10.0.2.15",
        "phase": "Running",
        "podIP": "172.17.0.7",
        "qosClass": "BestEffort",
        "startTime": "2020-03-13T03:38:57Z"
    }
}
`

var server2PodJsonData = `{
    "apiVersion": "v1",
    "kind": "Pod",
    "metadata": {
        "annotations": {
            "dubbo.io/annotation": "W3siayI6Ii9kdWJibyIsInYiOiIifSx7ImsiOiIvZHViYm8vY29tLmlrdXJlbnRvLnVzZXIuVXNlclByb3ZpZGVyIiwidiI6IiJ9LHsiayI6Ii9kdWJiby9jb20uaWt1cmVudG8udXNlci5Vc2VyUHJvdmlkZXIvcHJvdmlkZXJzIiwidiI6IiJ9LHsiayI6Ii9kdWJiby9jb20uaWt1cmVudG8udXNlci5Vc2VyUHJvdmlkZXIvcHJvdmlkZXJzL2R1YmJvJTNBJTJGJTJGMTcyLjE3LjAuNiUzQTIwMDAwJTJGVXNlclByb3ZpZGVyJTNGYWNjZXNzbG9nJTNEJTI2YW55aG9zdCUzRHRydWUlMjZhcHAudmVyc2lvbiUzRDAuMC4xJTI2YXBwbGljYXRpb24lM0RCRFRTZXJ2aWNlJTI2YmVhbi5uYW1lJTNEVXNlclByb3ZpZGVyJTI2Y2F0ZWdvcnklM0Rwcm92aWRlcnMlMjZjbHVzdGVyJTNEZmFpbG92ZXIlMjZkdWJibyUzRGR1YmJvLXByb3ZpZGVyLWdvbGFuZy0yLjYuMCUyNmVudmlyb25tZW50JTNEZGV2JTI2ZXhlY3V0ZS5saW1pdCUzRCUyNmV4ZWN1dGUubGltaXQucmVqZWN0ZWQuaGFuZGxlciUzRCUyNmdyb3VwJTNEJTI2aW50ZXJmYWNlJTNEY29tLmlrdXJlbnRvLnVzZXIuVXNlclByb3ZpZGVyJTI2aXAlM0QxNzIuMTcuMC42JTI2bG9hZGJhbGFuY2UlM0RyYW5kb20lMjZtZXRob2RzLkdldFVzZXIubG9hZGJhbGFuY2UlM0RyYW5kb20lMjZtZXRob2RzLkdldFVzZXIucmV0cmllcyUzRDElMjZtZXRob2RzLkdldFVzZXIudHBzLmxpbWl0LmludGVydmFsJTNEJTI2bWV0aG9kcy5HZXRVc2VyLnRwcy5saW1pdC5yYXRlJTNEJTI2bWV0aG9kcy5HZXRVc2VyLnRwcy5saW1pdC5zdHJhdGVneSUzRCUyNm1ldGhvZHMuR2V0VXNlci53ZWlnaHQlM0QwJTI2bW9kdWxlJTNEZHViYm9nbyUyQnVzZXItaW5mbyUyQnNlcnZlciUyNm5hbWUlM0RCRFRTZXJ2aWNlJTI2b3JnYW5pemF0aW9uJTNEaWt1cmVudG8uY29tJTI2b3duZXIlM0RaWCUyNnBpZCUzRDEwJTI2cmVnaXN0cnkucm9sZSUzRDMlMjZyZXRyaWVzJTNEJTI2c2VydmljZS5maWx0ZXIlM0RlY2hvJTI1MkN0b2tlbiUyNTJDYWNjZXNzbG9nJTI1MkN0cHMlMjUyQ2V4ZWN1dGUlMjZzaWRlJTNEcHJvdmlkZXIlMjZ0aW1lc3RhbXAlM0QxNTg0MDcwODA5JTI2dHBzLmxpbWl0LmludGVydmFsJTNEJTI2dHBzLmxpbWl0LnJhdGUlM0QlMjZ0cHMubGltaXQucmVqZWN0ZWQuaGFuZGxlciUzRCUyNnRwcy5saW1pdC5zdHJhdGVneSUzRCUyNnRwcy5saW1pdGVyJTNEJTI2dmVyc2lvbiUzRCUyNndhcm11cCUzRDEwMCIsInYiOiIifV0="
        },
        "creationTimestamp": "2020-03-13T03:38:57Z",
        "generateName": "server-5b8f9f85c6-",
        "labels": {
            "dubbo.io/label": "dubbo.io-value",
            "pod-template-hash": "5b8f9f85c6",
            "role": "server"
        },
        "name": "server-5b8f9f85c6-xk5md",
        "namespace": "default",
        "ownerReferences": [
            {
                "apiVersion": "apps/v1",
                "blockOwnerDeletion": true,
                "controller": true,
                "kind": "ReplicaSet",
                "name": "server-5b8f9f85c6",
                "uid": "65e9d2b0-f286-4b21-ac31-260f1412556b"
            }
        ],
        "resourceVersion": "2449667",
        "selfLink": "/api/v1/namespaces/default/pods/server-5b8f9f85c6-xk5md",
        "uid": "9e59e164-6620-473b-a983-472ebc1120e9"
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
                "image": "registry.cn-hangzhou.aliyuncs.com/scottwang/dubbogo-server",
                "imagePullPolicy": "Always",
                "name": "server",
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
                "lastTransitionTime": "2020-03-13T03:40:09Z",
                "status": "True",
                "type": "Ready"
            },
            {
                "lastProbeTime": null,
                "lastTransitionTime": "2020-03-13T03:40:09Z",
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
                "containerID": "docker://442dc055392cc720b6a6eb0bc4105f13ea86e63cbdced5c83f15463bc61add76",
                "image": "registry.cn-hangzhou.aliyuncs.com/scottwang/dubbogo-server:latest",
                "imageID": "docker-pullable://registry.cn-hangzhou.aliyuncs.com/scottwang/dubbogo-server@sha256:60654ddba3a16ca3de52c8e30650a4c1d8b2ed8f8542af489b7a5a459e46fe6b",
                "lastState": {},
                "name": "server",
                "ready": true,
                "restartCount": 0,
                "state": {
                    "running": {
                        "startedAt": "2020-03-13T03:40:09Z"
                    }
                }
            }
        ],
        "hostIP": "10.0.2.15",
        "phase": "Running",
        "podIP": "172.17.0.6",
        "qosClass": "BestEffort",
        "startTime": "2020-03-13T03:38:57Z"
    }
}`

type KubernetesClientTestSuite struct {
	suite.Suite

	client *Client

	currentPod     v1.Pod
	fakeServerPod1 v1.Pod
	fakeServerPod2 v1.Pod
}

func (s *KubernetesClientTestSuite) SetupSuite() {

	t := s.T()

	// 1. install test data
	if err := json.Unmarshal([]byte(clientPodJsonData), &s.currentPod); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(server1PodJsonData), &s.fakeServerPod1); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(server2PodJsonData), &s.fakeServerPod2); err != nil {
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

func (s *KubernetesClientTestSuite) TearDownSuite() {
	s.client.Close()
	os.Unsetenv(podNameKey)
	os.Unsetenv(nameSpaceKey)
}

func (s *KubernetesClientTestSuite) SetupTest() {

	t := s.T()
	var err error
	s.client, err = newMockClient(s.currentPod.GetNamespace(), func() (kubernetes.Interface, error) {

		out := fake.NewSimpleClientset()

		// mock current pod
		if _, err := out.CoreV1().Pods(s.currentPod.GetNamespace()).Create(&s.currentPod); err != nil {
			return nil, errors.WithMessage(err, "mock current pod ")
		}
		return out, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (s *KubernetesClientTestSuite) TestClientValid() {

	t := s.T()

	if s.client.Valid() != true {
		t.Fatal("client is not valid")
	}
	s.client.Close()

	if s.client.Valid() != false {
		t.Fatal("client is valid")
	}
}

func (s *KubernetesClientTestSuite) TestClientDone() {

	t := s.T()

	go func() {
		time.Sleep(time.Second)
		s.client.Close()
	}()

	<-s.client.Done()

	if s.client.Valid() == true {
		t.Fatal("client should be invalid then")
	}
}

func (s *KubernetesClientTestSuite) TestClientCreateKV() {

	t := s.T()
	defer s.client.Close()

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v

		if err := s.client.Create(k, v); err != nil {
			t.Fatal(err)
		}

	}
}

func (s *KubernetesClientTestSuite) TestClientGetChildrenKVList() {

	t := s.T()
	defer s.client.Close()

	expect := make(map[string]string)
	got := make(map[string]string)

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v

		if strings.Contains(k, prefix) {
			expect[k] = v
		}

		if err := s.client.Create(k, v); err != nil {
			t.Fatal(err)
		}
	}

	kList, vList, err := s.client.GetChildren(prefix)
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

func (s *KubernetesClientTestSuite) TestClientWatch() {

	t := s.T()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {

		defer wg.Done()

		wc, err := s.client.WatchWithPrefix(prefix)
		if err != nil {
			t.Fatal(err)
		}

		for e := range wc {
			t.Logf("got event %v k %s v %s", e.EventType, e.Key, e.Value)
		}

	}()

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v

		if err := s.client.Create(k, v); err != nil {
			t.Fatal(err)
		}
	}

	s.client.Close()
	wg.Wait()
}

func TestKubernetesClient(t *testing.T) {
	suite.Run(t, new(KubernetesClientTestSuite))
}
