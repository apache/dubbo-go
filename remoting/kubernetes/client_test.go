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
var clientPodListJsonData = `
{
    "apiVersion": "v1",
    "items": [
        {
            "apiVersion": "v1",
            "kind": "Pod",
            "metadata": {
                "annotations": {
                    "dubbo.io/annotation": "W3siayI6Ii9kdWJiby9vcmcuYXBhY2hlLmR1YmJvLnF1aWNrc3RhcnQuc2FtcGxlcy5Vc2VyUHJvdmlkZXIvY29uc3VtZXJzIiwidiI6IiJ9LHsiayI6Ii9kdWJiby9vcmcuYXBhY2hlLmR1YmJvLnF1aWNrc3RhcnQuc2FtcGxlcy5Vc2VyUHJvdmlkZXIvY29uc3VtZXJzL2NvbnN1bWVyJTNBJTJGJTJGMTAuMjQ0LjAuNzglMkZvcmcuYXBhY2hlLmR1YmJvLnF1aWNrc3RhcnQuc2FtcGxlcy5Vc2VyUHJvdmlkZXIlM0ZhcHAudmVyc2lvbiUzRDMuMC4wJTI2YXBwbGljYXRpb24lM0RkdWJiby5pbyUyNmFzeW5jJTNEZmFsc2UlMjZiZWFuLm5hbWUlM0RVc2VyUHJvdmlkZXJDbGllbnRJbXBsJTI2Y2x1c3RlciUzRGZhaWxvdmVyJTI2Y29uZmlnLnRyYWNpbmclM0QlMjZlbnZpcm9ubWVudCUzRGRldiUyNmdlbmVyaWMlM0QlMjZncm91cCUzRCUyNmludGVyZmFjZSUzRG9yZy5hcGFjaGUuZHViYm8ucXVpY2tzdGFydC5zYW1wbGVzLlVzZXJQcm92aWRlciUyNmxvYWRiYWxhbmNlJTNEJTI2bWV0YWRhdGEtdHlwZSUzRGxvY2FsJTI2bW9kdWxlJTNEc2FtcGxlJTI2bmFtZSUzRGR1YmJvLmlvJTI2b3JnYW5pemF0aW9uJTNEZHViYm8tZ28lMjZvd25lciUzRGR1YmJvLWdvJTI2cHJvdG9jb2wlM0R0cmklMjZwcm92aWRlZC1ieSUzRCUyNnJlZmVyZW5jZS5maWx0ZXIlM0Rjc2h1dGRvd24lMjZyZWdpc3RyeS5yb2xlJTNEMCUyNnJlbGVhc2UlM0RkdWJiby1nb2xhbmctMy4wLjAlMjZyZXRyaWVzJTNEJTI2c2VyaWFsaXphdGlvbiUzRCUyNnNpZGUlM0Rjb25zdW1lciUyNnN0aWNreSUzRGZhbHNlJTI2dGltZXN0YW1wJTNEMTY0ODEyNTQyNCUyNnZlcnNpb24lM0QiLCJ2IjoiIn0seyJrIjoiL2R1YmJvL29yZy5hcGFjaGUuZHViYm8ucXVpY2tzdGFydC5zYW1wbGVzLlVzZXJQcm92aWRlci9jb25zdW1lcnMiLCJ2IjoiIn0seyJrIjoiL2R1YmJvL29yZy5hcGFjaGUuZHViYm8ucXVpY2tzdGFydC5zYW1wbGVzLlVzZXJQcm92aWRlci9jb25zdW1lcnMvY29uc3VtZXIlM0ElMkYlMkYxMC4yNDQuMC43OCUyRm9yZy5hcGFjaGUuZHViYm8ucXVpY2tzdGFydC5zYW1wbGVzLlVzZXJQcm92aWRlciUzRmFwcC52ZXJzaW9uJTNEMy4wLjAlMjZhcHBsaWNhdGlvbiUzRGR1YmJvLmlvJTI2YXN5bmMlM0RmYWxzZSUyNmJlYW4ubmFtZSUzRFVzZXJQcm92aWRlckNsaWVudEltcGwlMjZjbHVzdGVyJTNEZmFpbG92ZXIlMjZjb25maWcudHJhY2luZyUzRCUyNmVudmlyb25tZW50JTNEZGV2JTI2Z2VuZXJpYyUzRCUyNmdyb3VwJTNEJTI2aW50ZXJmYWNlJTNEb3JnLmFwYWNoZS5kdWJiby5xdWlja3N0YXJ0LnNhbXBsZXMuVXNlclByb3ZpZGVyJTI2bG9hZGJhbGFuY2UlM0QlMjZtZXRhZGF0YS10eXBlJTNEbG9jYWwlMjZtb2R1bGUlM0RzYW1wbGUlMjZuYW1lJTNEZHViYm8uaW8lMjZvcmdhbml6YXRpb24lM0RkdWJiby1nbyUyNm93bmVyJTNEZHViYm8tZ28lMjZwcm90b2NvbCUzRHRyaSUyNnByb3ZpZGVkLWJ5JTNEJTI2cmVmZXJlbmNlLmZpbHRlciUzRGNzaHV0ZG93biUyNnJlZ2lzdHJ5LnJvbGUlM0QwJTI2cmVsZWFzZSUzRGR1YmJvLWdvbGFuZy0zLjAuMCUyNnJldHJpZXMlM0QlMjZzZXJpYWxpemF0aW9uJTNEJTI2c2lkZSUzRGNvbnN1bWVyJTI2c3RpY2t5JTNEZmFsc2UlMjZ0aW1lc3RhbXAlM0QxNjQ4MTI1NDc5JTI2dmVyc2lvbiUzRCIsInYiOiIifV0=",
                    "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"annotations\":{},\"name\":\"client\",\"namespace\":\"default\"},\"spec\":{\"containers\":[{\"env\":[{\"name\":\"NAMESPACE\",\"valueFrom\":{\"fieldRef\":{\"fieldPath\":\"metadata.namespace\"}}},{\"name\":\"DUBBO_NAMESPACE\",\"valueFrom\":{\"fieldRef\":{\"fieldPath\":\"metadata.namespace\"}}}],\"image\":\"registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-client:0.0.16\",\"imagePullPolicy\":\"Always\",\"name\":\"client\"}],\"restartPolicy\":\"OnFailure\",\"serviceAccountName\":\"dubbo-sa\"}}\n"
                },
                "creationTimestamp": "2022-03-24T12:33:48Z",
                "labels": {
                    "dubbo.io/label": "dubbo.io.consumer"
                },
                "name": "client",
                "namespace": "default",
                "resourceVersion": "217609",
                "uid": "2857b7cf-f32b-4dd5-bae0-c45a743f97e7"
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
                            },
                            {
                                "name": "DUBBO_NAMESPACE",
                                "valueFrom": {
                                    "fieldRef": {
                                        "apiVersion": "v1",
                                        "fieldPath": "metadata.namespace"
                                    }
                                }
                            }
                        ],
                        "image": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-client:0.0.16",
                        "imagePullPolicy": "Always",
                        "name": "client",
                        "resources": {},
                        "terminationMessagePath": "/dev/termination-log",
                        "terminationMessagePolicy": "File",
                        "volumeMounts": [
                            {
                                "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount",
                                "name": "kube-api-access-8bs9r",
                                "readOnly": true
                            }
                        ]
                    }
                ],
                "dnsPolicy": "ClusterFirst",
                "enableServiceLinks": true,
                "nodeName": "kind-control-plane",
                "preemptionPolicy": "PreemptLowerPriority",
                "priority": 0,
                "restartPolicy": "OnFailure",
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
                        "name": "kube-api-access-8bs9r",
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
                        "lastTransitionTime": "2022-03-24T12:33:48Z",
                        "reason": "PodCompleted",
                        "status": "True",
                        "type": "Initialized"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-24T12:38:13Z",
                        "reason": "PodCompleted",
                        "status": "False",
                        "type": "Ready"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-24T12:38:13Z",
                        "reason": "PodCompleted",
                        "status": "False",
                        "type": "ContainersReady"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-24T12:33:48Z",
                        "status": "True",
                        "type": "PodScheduled"
                    }
                ],
                "containerStatuses": [
                    {
                        "containerID": "containerd://f213071de484ac6251c0f19d5fb303bf0e534efa40b64fe4575423fc0c5185f2",
                        "image": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-client:0.0.16",
                        "imageID": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-client@sha256:ff85d66f58a7b9c3355516ca7ad3495b4f07a8e26ea85be52614830f001add4e",
                        "lastState": {},
                        "name": "client",
                        "ready": false,
                        "restartCount": 1,
                        "started": false,
                        "state": {
                            "terminated": {
                                "containerID": "containerd://f213071de484ac6251c0f19d5fb303bf0e534efa40b64fe4575423fc0c5185f2",
                                "exitCode": 0,
                                "finishedAt": "2022-03-24T12:38:13Z",
                                "reason": "Completed",
                                "startedAt": "2022-03-24T12:37:59Z"
                            }
                        }
                    }
                ],
                "hostIP": "172.18.0.2",
                "phase": "Succeeded",
                "podIP": "10.244.0.78",
                "podIPs": [
                    {
                        "ip": "10.244.0.78"
                    }
                ],
                "qosClass": "BestEffort",
                "startTime": "2022-03-24T12:33:48Z"
            }
        },
        {
            "apiVersion": "v1",
            "kind": "Pod",
            "metadata": {
                "annotations": {
                    "dubbo.io/annotation": "W3siayI6Ii9kdWJiby9ncnBjLnJlZmxlY3Rpb24udjFhbHBoYS5TZXJ2ZXJSZWZsZWN0aW9uL3Byb3ZpZGVycyIsInYiOiIifSx7ImsiOiIvZHViYm8vZ3JwYy5yZWZsZWN0aW9uLnYxYWxwaGEuU2VydmVyUmVmbGVjdGlvbi9wcm92aWRlcnMvdHJpJTNBJTJGJTJGMTAuMjQ0LjAuODElM0EyMDAwMSUyRmdycGMucmVmbGVjdGlvbi52MWFscGhhLlNlcnZlclJlZmxlY3Rpb24lM0Zhbnlob3N0JTNEdHJ1ZSUyNmFwcC52ZXJzaW9uJTNEMy4wLjAlMjZhcHBsaWNhdGlvbiUzRGR1YmJvLmlvJTI2YmVhbi5uYW1lJTNEWFhYX3NlcnZlclJlZmxlY3Rpb25TZXJ2ZXIlMjZjbHVzdGVyJTNEZmFpbG92ZXIlMjZlbnZpcm9ubWVudCUzRGRldiUyNmV4cG9ydCUzRHRydWUlMjZpbnRlcmZhY2UlM0RncnBjLnJlZmxlY3Rpb24udjFhbHBoYS5TZXJ2ZXJSZWZsZWN0aW9uJTI2bG9hZGJhbGFuY2UlM0RyYW5kb20lMjZtZXNzYWdlX3NpemUlM0Q0JTI2bWV0YWRhdGEtdHlwZSUzRGxvY2FsJTI2bWV0aG9kcyUzRFNlcnZlclJlZmxlY3Rpb25JbmZvJTI2bW9kdWxlJTNEc2FtcGxlJTI2bmFtZSUzRGR1YmJvLmlvJTI2b3JnYW5pemF0aW9uJTNEZHViYm8tZ28lMjZvd25lciUzRGR1YmJvLWdvJTI2cGlkJTNEMTMlMjZyZWdpc3RyeSUzRGt1YmVybmV0ZXMlMjZyZWdpc3RyeS5yb2xlJTNEMyUyNnJlbGVhc2UlM0RkdWJiby1nb2xhbmctMy4wLjAlMjZzZXJ2aWNlLmZpbHRlciUzRGVjaG8lMkNtZXRyaWNzJTJDdG9rZW4lMkNhY2Nlc3Nsb2clMkN0cHMlMkNnZW5lcmljX3NlcnZpY2UlMkNleGVjdXRlJTJDcHNodXRkb3duJTI2c2lkZSUzRHByb3ZpZGVyJTI2dGltZXN0YW1wJTNEMTY0ODEyNTQ5MiIsInYiOiIifSx7ImsiOiIvZHViYm8vb3JnLmFwYWNoZS5kdWJiby5xdWlja3N0YXJ0LnNhbXBsZXMuVXNlclByb3ZpZGVyL3Byb3ZpZGVycyIsInYiOiIifSx7ImsiOiIvZHViYm8vb3JnLmFwYWNoZS5kdWJiby5xdWlja3N0YXJ0LnNhbXBsZXMuVXNlclByb3ZpZGVyL3Byb3ZpZGVycy90cmklM0ElMkYlMkYxMC4yNDQuMC44MSUzQTIwMDAxJTJGb3JnLmFwYWNoZS5kdWJiby5xdWlja3N0YXJ0LnNhbXBsZXMuVXNlclByb3ZpZGVyJTNGYW55aG9zdCUzRHRydWUlMjZhcHAudmVyc2lvbiUzRDMuMC4wJTI2YXBwbGljYXRpb24lM0RkdWJiby5pbyUyNmJlYW4ubmFtZSUzREdyZWV0ZXJQcm92aWRlciUyNmNsdXN0ZXIlM0RmYWlsb3ZlciUyNmVudmlyb25tZW50JTNEZGV2JTI2ZXhwb3J0JTNEdHJ1ZSUyNmludGVyZmFjZSUzRG9yZy5hcGFjaGUuZHViYm8ucXVpY2tzdGFydC5zYW1wbGVzLlVzZXJQcm92aWRlciUyNmxvYWRiYWxhbmNlJTNEcmFuZG9tJTI2bWVzc2FnZV9zaXplJTNENCUyNm1ldGFkYXRhLXR5cGUlM0Rsb2NhbCUyNm1ldGhvZHMlM0RTYXlIZWxsbyUyQ1NheUhlbGxvU3RyZWFtJTI2bW9kdWxlJTNEc2FtcGxlJTI2bmFtZSUzRGR1YmJvLmlvJTI2b3JnYW5pemF0aW9uJTNEZHViYm8tZ28lMjZvd25lciUzRGR1YmJvLWdvJTI2cGlkJTNEMTMlMjZyZWdpc3RyeSUzRGt1YmVybmV0ZXMlMjZyZWdpc3RyeS5yb2xlJTNEMyUyNnJlbGVhc2UlM0RkdWJiby1nb2xhbmctMy4wLjAlMjZzZXJ2aWNlLmZpbHRlciUzRGVjaG8lMkNtZXRyaWNzJTJDdG9rZW4lMkNhY2Nlc3Nsb2clMkN0cHMlMkNnZW5lcmljX3NlcnZpY2UlMkNleGVjdXRlJTJDcHNodXRkb3duJTI2c2lkZSUzRHByb3ZpZGVyJTI2dGltZXN0YW1wJTNEMTY0ODEyNTQ5MiIsInYiOiIifV0="
                },
                "creationTimestamp": "2022-03-24T12:38:10Z",
                "generateName": "server-748b6d68f9-",
                "labels": {
                    "dubbo.io/label": "dubbo.io.provider",
                    "pod-template-hash": "748b6d68f9",
                    "role": "server"
                },
                "name": "server-748b6d68f9-nmf5f",
                "namespace": "default",
                "ownerReferences": [
                    {
                        "apiVersion": "apps/v1",
                        "blockOwnerDeletion": true,
                        "controller": true,
                        "kind": "ReplicaSet",
                        "name": "server-748b6d68f9",
                        "uid": "033eb35b-c557-47a3-b48e-1fbef43cce3f"
                    }
                ],
                "resourceVersion": "217596",
                "uid": "d16aa366-d435-4a83-aee4-dd86a3720af7"
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
                            },
                            {
                                "name": "DUBBO_NAMESPACE",
                                "valueFrom": {
                                    "fieldRef": {
                                        "apiVersion": "v1",
                                        "fieldPath": "metadata.namespace"
                                    }
                                }
                            }
                        ],
                        "image": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-server:0.0.16",
                        "imagePullPolicy": "Always",
                        "name": "server",
                        "resources": {},
                        "terminationMessagePath": "/dev/termination-log",
                        "terminationMessagePolicy": "File",
                        "volumeMounts": [
                            {
                                "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount",
                                "name": "kube-api-access-5js54",
                                "readOnly": true
                            }
                        ]
                    }
                ],
                "dnsPolicy": "ClusterFirst",
                "enableServiceLinks": true,
                "nodeName": "kind-control-plane",
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
                        "name": "kube-api-access-5js54",
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
                        "lastTransitionTime": "2022-03-24T12:38:10Z",
                        "status": "True",
                        "type": "Initialized"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-24T12:38:12Z",
                        "status": "True",
                        "type": "Ready"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-24T12:38:12Z",
                        "status": "True",
                        "type": "ContainersReady"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-24T12:38:10Z",
                        "status": "True",
                        "type": "PodScheduled"
                    }
                ],
                "containerStatuses": [
                    {
                        "containerID": "containerd://89b82ded850328e4c3d08cfc17456193708f25cbad86591d259d6796f9984308",
                        "image": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-server:0.0.16",
                        "imageID": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-server@sha256:dd98f174ea1393f860aef3d16b6f326aae296455765211ba0954b948d0f70e9e",
                        "lastState": {},
                        "name": "server",
                        "ready": true,
                        "restartCount": 0,
                        "started": true,
                        "state": {
                            "running": {
                                "startedAt": "2022-03-24T12:38:12Z"
                            }
                        }
                    }
                ],
                "hostIP": "172.18.0.2",
                "phase": "Running",
                "podIP": "10.244.0.81",
                "podIPs": [
                    {
                        "ip": "10.244.0.81"
                    }
                ],
                "qosClass": "BestEffort",
                "startTime": "2022-03-24T12:38:10Z"
            }
        },
        {
            "apiVersion": "v1",
            "kind": "Pod",
            "metadata": {
                "annotations": {
                    "dubbo.io/annotation": "W3siayI6Ii9kdWJiby9vcmcuYXBhY2hlLmR1YmJvLnF1aWNrc3RhcnQuc2FtcGxlcy5Vc2VyUHJvdmlkZXIvcHJvdmlkZXJzIiwidiI6IiJ9LHsiayI6Ii9kdWJiby9vcmcuYXBhY2hlLmR1YmJvLnF1aWNrc3RhcnQuc2FtcGxlcy5Vc2VyUHJvdmlkZXIvcHJvdmlkZXJzL3RyaSUzQSUyRiUyRjEwLjI0NC4wLjgyJTNBMjAwMDElMkZvcmcuYXBhY2hlLmR1YmJvLnF1aWNrc3RhcnQuc2FtcGxlcy5Vc2VyUHJvdmlkZXIlM0Zhbnlob3N0JTNEdHJ1ZSUyNmFwcC52ZXJzaW9uJTNEMy4wLjAlMjZhcHBsaWNhdGlvbiUzRGR1YmJvLmlvJTI2YmVhbi5uYW1lJTNER3JlZXRlclByb3ZpZGVyJTI2Y2x1c3RlciUzRGZhaWxvdmVyJTI2ZW52aXJvbm1lbnQlM0RkZXYlMjZleHBvcnQlM0R0cnVlJTI2aW50ZXJmYWNlJTNEb3JnLmFwYWNoZS5kdWJiby5xdWlja3N0YXJ0LnNhbXBsZXMuVXNlclByb3ZpZGVyJTI2bG9hZGJhbGFuY2UlM0RyYW5kb20lMjZtZXNzYWdlX3NpemUlM0Q0JTI2bWV0YWRhdGEtdHlwZSUzRGxvY2FsJTI2bWV0aG9kcyUzRFNheUhlbGxvJTJDU2F5SGVsbG9TdHJlYW0lMjZtb2R1bGUlM0RzYW1wbGUlMjZuYW1lJTNEZHViYm8uaW8lMjZvcmdhbml6YXRpb24lM0RkdWJiby1nbyUyNm93bmVyJTNEZHViYm8tZ28lMjZwaWQlM0QxNiUyNnJlZ2lzdHJ5JTNEa3ViZXJuZXRlcyUyNnJlZ2lzdHJ5LnJvbGUlM0QzJTI2cmVsZWFzZSUzRGR1YmJvLWdvbGFuZy0zLjAuMCUyNnNlcnZpY2UuZmlsdGVyJTNEZWNobyUyQ21ldHJpY3MlMkN0b2tlbiUyQ2FjY2Vzc2xvZyUyQ3RwcyUyQ2dlbmVyaWNfc2VydmljZSUyQ2V4ZWN1dGUlMkNwc2h1dGRvd24lMjZzaWRlJTNEcHJvdmlkZXIlMjZ0aW1lc3RhbXAlM0QxNjQ4MTI1NDk0IiwidiI6IiJ9LHsiayI6Ii9kdWJiby9ncnBjLnJlZmxlY3Rpb24udjFhbHBoYS5TZXJ2ZXJSZWZsZWN0aW9uL3Byb3ZpZGVycyIsInYiOiIifSx7ImsiOiIvZHViYm8vZ3JwYy5yZWZsZWN0aW9uLnYxYWxwaGEuU2VydmVyUmVmbGVjdGlvbi9wcm92aWRlcnMvdHJpJTNBJTJGJTJGMTAuMjQ0LjAuODIlM0EyMDAwMSUyRmdycGMucmVmbGVjdGlvbi52MWFscGhhLlNlcnZlclJlZmxlY3Rpb24lM0Zhbnlob3N0JTNEdHJ1ZSUyNmFwcC52ZXJzaW9uJTNEMy4wLjAlMjZhcHBsaWNhdGlvbiUzRGR1YmJvLmlvJTI2YmVhbi5uYW1lJTNEWFhYX3NlcnZlclJlZmxlY3Rpb25TZXJ2ZXIlMjZjbHVzdGVyJTNEZmFpbG92ZXIlMjZlbnZpcm9ubWVudCUzRGRldiUyNmV4cG9ydCUzRHRydWUlMjZpbnRlcmZhY2UlM0RncnBjLnJlZmxlY3Rpb24udjFhbHBoYS5TZXJ2ZXJSZWZsZWN0aW9uJTI2bG9hZGJhbGFuY2UlM0RyYW5kb20lMjZtZXNzYWdlX3NpemUlM0Q0JTI2bWV0YWRhdGEtdHlwZSUzRGxvY2FsJTI2bWV0aG9kcyUzRFNlcnZlclJlZmxlY3Rpb25JbmZvJTI2bW9kdWxlJTNEc2FtcGxlJTI2bmFtZSUzRGR1YmJvLmlvJTI2b3JnYW5pemF0aW9uJTNEZHViYm8tZ28lMjZvd25lciUzRGR1YmJvLWdvJTI2cGlkJTNEMTYlMjZyZWdpc3RyeSUzRGt1YmVybmV0ZXMlMjZyZWdpc3RyeS5yb2xlJTNEMyUyNnJlbGVhc2UlM0RkdWJiby1nb2xhbmctMy4wLjAlMjZzZXJ2aWNlLmZpbHRlciUzRGVjaG8lMkNtZXRyaWNzJTJDdG9rZW4lMkNhY2Nlc3Nsb2clMkN0cHMlMkNnZW5lcmljX3NlcnZpY2UlMkNleGVjdXRlJTJDcHNodXRkb3duJTI2c2lkZSUzRHByb3ZpZGVyJTI2dGltZXN0YW1wJTNEMTY0ODEyNTQ5NCIsInYiOiIifV0="
                },
                "creationTimestamp": "2022-03-24T12:38:10Z",
                "generateName": "server-748b6d68f9-",
                "labels": {
                    "dubbo.io/label": "dubbo.io.provider",
                    "pod-template-hash": "748b6d68f9",
                    "role": "server"
                },
                "name": "server-748b6d68f9-vvb24",
                "namespace": "default",
                "ownerReferences": [
                    {
                        "apiVersion": "apps/v1",
                        "blockOwnerDeletion": true,
                        "controller": true,
                        "kind": "ReplicaSet",
                        "name": "server-748b6d68f9",
                        "uid": "033eb35b-c557-47a3-b48e-1fbef43cce3f"
                    }
                ],
                "resourceVersion": "217610",
                "uid": "0b3d2de3-fa47-4d44-9812-dabf717531b9"
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
                            },
                            {
                                "name": "DUBBO_NAMESPACE",
                                "valueFrom": {
                                    "fieldRef": {
                                        "apiVersion": "v1",
                                        "fieldPath": "metadata.namespace"
                                    }
                                }
                            }
                        ],
                        "image": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-server:0.0.16",
                        "imagePullPolicy": "Always",
                        "name": "server",
                        "resources": {},
                        "terminationMessagePath": "/dev/termination-log",
                        "terminationMessagePolicy": "File",
                        "volumeMounts": [
                            {
                                "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount",
                                "name": "kube-api-access-5tnn9",
                                "readOnly": true
                            }
                        ]
                    }
                ],
                "dnsPolicy": "ClusterFirst",
                "enableServiceLinks": true,
                "nodeName": "kind-control-plane",
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
                        "name": "kube-api-access-5tnn9",
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
                        "lastTransitionTime": "2022-03-24T12:38:10Z",
                        "status": "True",
                        "type": "Initialized"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-24T12:38:14Z",
                        "status": "True",
                        "type": "Ready"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-24T12:38:14Z",
                        "status": "True",
                        "type": "ContainersReady"
                    },
                    {
                        "lastProbeTime": null,
                        "lastTransitionTime": "2022-03-24T12:38:10Z",
                        "status": "True",
                        "type": "PodScheduled"
                    }
                ],
                "containerStatuses": [
                    {
                        "containerID": "containerd://c71a0e2c992ada96d3b4138ac4b0520ae2c93f3d41deda89016932925a6047cc",
                        "image": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-server:0.0.16",
                        "imageID": "registry.cn-hangzhou.aliyuncs.com/amrom/dubbo-go-server@sha256:dd98f174ea1393f860aef3d16b6f326aae296455765211ba0954b948d0f70e9e",
                        "lastState": {},
                        "name": "server",
                        "ready": true,
                        "restartCount": 0,
                        "started": true,
                        "state": {
                            "running": {
                                "startedAt": "2022-03-24T12:38:14Z"
                            }
                        }
                    }
                ],
                "hostIP": "172.18.0.2",
                "phase": "Running",
                "podIP": "10.244.0.82",
                "podIPs": [
                    {
                        "ip": "10.244.0.82"
                    }
                ],
                "qosClass": "BestEffort",
                "startTime": "2022-03-24T12:38:10Z"
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
			t.Error(err)
			return
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
