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

package client

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type newClientCase struct {
	desc   string
	opts   []ClientOption
	verify func(t *testing.T, cli *Client, err error)
}

func processNewClientCases(t *testing.T, cases []newClientCase) {
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			cli, err := NewClient(c.opts...)
			c.verify(t, cli, err)
		})
	}
}

// ---------- ClientOption Testing ----------

// todo: verify
func TestWithClientURL(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "normal address",
			opts: []ClientOption{
				WithClientURL("127.0.0.1:20000"),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "127.0.0.1:20000", cli.cliOpts.overallReference.URL)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientCheck(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config check",
			opts: []ClientOption{
				WithClientCheck(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, true, cli.cliOpts.Consumer.Check)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientFilter(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config normal filter",
			opts: []ClientOption{
				WithClientFilter("test_filter"),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "test_filter", cli.cliOpts.Consumer.Filter)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientRegistryIDs(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config normal ids",
			opts: []ClientOption{
				WithClientRegistryIDs("zk", "nacos"),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, []string{"zk", "nacos"}, cli.cliOpts.Consumer.RegistryIDs)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientRegistry(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config registry without setting id explicitly",
			opts: []ClientOption{
				WithClientRegistry(
					registry.WithZookeeper(),
					registry.WithAddress("127.0.0.1:2181"),
				),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				reg, ok := cli.cliOpts.Registries[constant.ZookeeperKey]
				assert.True(t, ok)
				assert.Equal(t, "127.0.0.1:2181", reg.Address)
				regCompat, ok := cli.cliOpts.registriesCompat[constant.ZookeeperKey]
				assert.True(t, ok)
				assert.Equal(t, "127.0.0.1:2181", regCompat.Address)
			},
		},
		{
			desc: "config registry without setting id",
			opts: []ClientOption{
				WithClientRegistry(
					registry.WithID("zk"),
					registry.WithZookeeper(),
					registry.WithAddress("127.0.0.1:2181"),
				),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				reg, ok := cli.cliOpts.Registries["zk"]
				assert.True(t, ok)
				assert.Equal(t, "127.0.0.1:2181", reg.Address)
				regCompat, ok := cli.cliOpts.registriesCompat["zk"]
				assert.True(t, ok)
				assert.Equal(t, "127.0.0.1:2181", regCompat.Address)
			},
		},
		{
			desc: "config multiple registries with setting RegistryIds",
			opts: []ClientOption{
				WithClientRegistry(
					registry.WithZookeeper(),
					registry.WithAddress("127.0.0.1:2181"),
				),
				WithClientRegistry(
					registry.WithID("nacos_test"),
					registry.WithNacos(),
					registry.WithAddress("127.0.0.1:8848"),
				),
				WithClientRegistryIDs("nacos_test"),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				zkReg, ok := cli.cliOpts.Registries[constant.ZookeeperKey]
				assert.True(t, ok)
				assert.Equal(t, "127.0.0.1:2181", zkReg.Address)
				ncReg, ok := cli.cliOpts.Registries["nacos_test"]
				assert.True(t, ok)
				assert.Equal(t, "127.0.0.1:8848", ncReg.Address)

				_, ok = cli.cliOpts.registriesCompat[constant.ZookeeperKey]
				assert.False(t, ok)
				ncCompat, ok := cli.cliOpts.registriesCompat["nacos_test"]
				assert.True(t, ok)
				assert.Equal(t, "127.0.0.1:8848", ncCompat.Address)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientShutdown(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config shutdown",
			opts: []ClientOption{
				WithClientShutdown(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				// we do not verify the internal fields of Shutdown since graceful_shutdown module is in charge of it
				assert.NotNil(t, cli.cliOpts.Shutdown)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientCluster(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "default Cluster strategy",
			opts: []ClientOption{},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyFailover, cli.cliOpts.overallReference.Cluster)
			},
		},
		{
			desc: "config Available Cluster strategy",
			opts: []ClientOption{
				WithClientClusterAvailable(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyAvailable, cli.cliOpts.overallReference.Cluster)
			},
		},
		{
			desc: "config Broadcast Cluster strategy",
			opts: []ClientOption{
				WithClientClusterBroadcast(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyBroadcast, cli.cliOpts.overallReference.Cluster)
			},
		},
		{
			desc: "config FailBack Cluster strategy",
			opts: []ClientOption{
				WithClientClusterFailBack(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyFailback, cli.cliOpts.overallReference.Cluster)
			},
		},
		{
			desc: "config FailFast Cluster strategy",
			opts: []ClientOption{
				WithClientClusterFailFast(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyFailfast, cli.cliOpts.overallReference.Cluster)
			},
		},
		{
			desc: "config FailOver Cluster strategy",
			opts: []ClientOption{
				WithClientClusterFailOver(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyFailover, cli.cliOpts.overallReference.Cluster)
			},
		},
		{
			desc: "config FailSafe Cluster strategy",
			opts: []ClientOption{
				WithClientClusterFailSafe(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyFailsafe, cli.cliOpts.overallReference.Cluster)
			},
		},
		{
			desc: "config Forking Cluster strategy",
			opts: []ClientOption{
				WithClientClusterForking(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyForking, cli.cliOpts.overallReference.Cluster)
			},
		},
		{
			desc: "config ZoneAware Cluster strategy",
			opts: []ClientOption{
				WithClientClusterZoneAware(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyZoneAware, cli.cliOpts.overallReference.Cluster)
			},
		},
		{
			desc: "config AdaptiveService Cluster strategy",
			opts: []ClientOption{
				WithClientClusterAdaptiveService(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyAdaptiveService, cli.cliOpts.overallReference.Cluster)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientLoadBalance(t *testing.T) {
	cases := []newClientCase{
		// todo(DMwangnima): think about default loadbalance strategy
		//{
		//	desc: "default Cluster strategy",
		//	opts: []ClientOption{},
		//	verify: func(t *testing.T, cli *Client, err error) {
		//		assert.Nil(t, err)
		//		assert.Equal(t, constant.ClusterKeyFailover, cli.cliOpts.overallReference.Cluster)
		//	},
		//},
		{
			desc: "config ConsistentHashing LoadBalance strategy",
			opts: []ClientOption{
				WithClientLoadBalanceConsistentHashing(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.LoadBalanceKeyConsistentHashing, cli.cliOpts.overallReference.Loadbalance)
			},
		},
		{
			desc: "config LeastActive LoadBalance strategy",
			opts: []ClientOption{
				WithClientLoadBalanceLeastActive(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.LoadBalanceKeyLeastActive, cli.cliOpts.overallReference.Loadbalance)
			},
		},
		{
			desc: "config Random LoadBalance strategy",
			opts: []ClientOption{
				WithClientLoadBalanceRandom(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.LoadBalanceKeyRandom, cli.cliOpts.overallReference.Loadbalance)
			},
		},
		{
			desc: "config RoundRobin LoadBalance strategy",
			opts: []ClientOption{
				WithClientLoadBalanceRoundRobin(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.LoadBalanceKeyRoundRobin, cli.cliOpts.overallReference.Loadbalance)
			},
		},
		{
			desc: "config P2C LoadBalance strategy",
			opts: []ClientOption{
				WithClientLoadBalanceP2C(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.LoadBalanceKeyP2C, cli.cliOpts.overallReference.Loadbalance)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientRetries(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config normal retries",
			opts: []ClientOption{
				WithClientRetries(3),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "3", cli.cliOpts.overallReference.Retries)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientGroup(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config normal group",
			opts: []ClientOption{
				WithClientGroup("test_group"),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "test_group", cli.cliOpts.overallReference.Group)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientVersion(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config normal version",
			opts: []ClientOption{
				WithClientVersion("test_version"),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "test_version", cli.cliOpts.overallReference.Version)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientSerialization(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "default Serialization",
			opts: []ClientOption{},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ProtobufSerialization, cli.cliOpts.overallReference.Serialization)
			},
		},
		{
			desc: "config JSON Serialization",
			opts: []ClientOption{
				WithClientSerializationJSON(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.JSONSerialization, cli.cliOpts.overallReference.Serialization)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientProvidedBy(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config normal ProvidedBy",
			opts: []ClientOption{
				WithClientProvidedBy("test_instance"),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "test_instance", cli.cliOpts.overallReference.ProvidedBy)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientParams(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config normal params",
			opts: []ClientOption{
				WithClientParams(map[string]string{
					"test_key": "test_val",
				}),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, map[string]string{"test_key": "test_val"}, cli.cliOpts.overallReference.Params)
			},
		},
		{
			desc: "config nil params",
			opts: []ClientOption{
				WithClientParams(nil),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Nil(t, cli.cliOpts.overallReference.Params)
			},
		},
		{
			desc: "config nil params with type information",
			opts: []ClientOption{
				WithClientParams((map[string]string)(nil)),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Nil(t, cli.cliOpts.overallReference.Params)
			},
		},
		{
			desc: "config params without key-val",
			opts: []ClientOption{
				WithClientParams(map[string]string{}),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Nil(t, cli.cliOpts.overallReference.Params)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientParam(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config normal param",
			opts: []ClientOption{
				WithClientParam("test_key", "test_val"),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, map[string]string{"test_key": "test_val"}, cli.cliOpts.overallReference.Params)
			},
		},
		{
			desc: "config normal param multiple times",
			opts: []ClientOption{
				WithClientParam("test_key", "test_val"),
				WithClientParam("test_key1", "test_val1"),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, map[string]string{"test_key": "test_val", "test_key1": "test_val1"}, cli.cliOpts.overallReference.Params)
			},
		},
		{
			desc: "config param with empty key",
			opts: []ClientOption{
				WithClientParam("", ""),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, map[string]string{"": ""}, cli.cliOpts.overallReference.Params)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientSticky(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config sticky",
			opts: []ClientOption{
				WithClientSticky(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.True(t, cli.cliOpts.overallReference.Sticky)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientProtocol(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "default Protocol",
			opts: []ClientOption{},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "tri", cli.cliOpts.Consumer.Protocol)
			},
		},
		{
			desc: "config Dubbo Protocol",
			opts: []ClientOption{
				WithClientProtocolDubbo(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.Dubbo, cli.cliOpts.Consumer.Protocol)
			},
		},
		{
			desc: "config Triple Protocol",
			opts: []ClientOption{
				WithClientProtocolTriple(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "tri", cli.cliOpts.Consumer.Protocol)
			},
		},
		{
			desc: "config JsonRPC Protocol",
			opts: []ClientOption{
				WithClientProtocolJsonRPC(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "jsonrpc", cli.cliOpts.Consumer.Protocol)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientRequestTimeout(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config normal RequestTimeout",
			opts: []ClientOption{
				WithClientRequestTimeout(6 * time.Second),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "6s", cli.cliOpts.Consumer.RequestTimeout)
			},
		},
		// todo(DMwangnima): consider whether this default timeout is ideal
		{
			desc: "default RequestTimeout",
			opts: []ClientOption{},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "3s", cli.cliOpts.Consumer.RequestTimeout)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientForceTag(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config ForceTag",
			opts: []ClientOption{
				WithClientForceTag(),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.True(t, cli.cliOpts.overallReference.ForceTag)
			},
		},
	}
	processNewClientCases(t, cases)
}

func TestWithClientMeshProviderPort(t *testing.T) {
	cases := []newClientCase{
		{
			desc: "config normal MeshProviderPort",
			opts: []ClientOption{
				WithClientMeshProviderPort(20001),
			},
			verify: func(t *testing.T, cli *Client, err error) {
				assert.Nil(t, err)
				assert.Equal(t, 20001, cli.cliOpts.overallReference.MeshProviderPort)
			},
		},
	}
	processNewClientCases(t, cases)
}

// ---------- ReferenceOption Testing ----------

type referenceOptionsInitCase struct {
	desc   string
	opts   []ReferenceOption
	verify func(t *testing.T, refOpts *ReferenceOptions, err error)
}

func processReferenceOptionsInitCases(t *testing.T, cases []referenceOptionsInitCase) {
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			defRefOpts := defaultReferenceOptions()
			err := defRefOpts.init(c.opts...)
			c.verify(t, defRefOpts, err)
		})
	}
}

func TestWithCheck(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config Check",
			opts: []ReferenceOption{
				WithCheck(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.True(t, *refOpts.Reference.Check)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithURL(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config normal URL",
			opts: []ReferenceOption{
				WithURL("127.0.0.1:20000"),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "127.0.0.1:20000", refOpts.Reference.URL)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithFilter(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config normal filter",
			opts: []ReferenceOption{
				WithFilter("test_filter"),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "test_filter", refOpts.Reference.Filter)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithRegistryIDs(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config normal ids",
			opts: []ReferenceOption{
				WithRegistryIDs("zk", "nacos"),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, []string{"zk", "nacos"}, refOpts.Reference.RegistryIDs)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithCluster(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "default Cluster strategy",
			opts: []ReferenceOption{},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyFailover, refOpts.Reference.Cluster)
			},
		},
		{
			desc: "config Available Cluster strategy",
			opts: []ReferenceOption{
				WithClusterAvailable(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyAvailable, refOpts.Reference.Cluster)
			},
		},
		{
			desc: "config Broadcast Cluster strategy",
			opts: []ReferenceOption{
				WithClusterBroadcast(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyBroadcast, refOpts.Reference.Cluster)
			},
		},
		{
			desc: "config FailBack Cluster strategy",
			opts: []ReferenceOption{
				WithClusterFailBack(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyFailback, refOpts.Reference.Cluster)
			},
		},
		{
			desc: "config FailFast Cluster strategy",
			opts: []ReferenceOption{
				WithClusterFailFast(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyFailfast, refOpts.Reference.Cluster)
			},
		},
		{
			desc: "config FailOver Cluster strategy",
			opts: []ReferenceOption{
				WithClusterFailOver(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyFailover, refOpts.Reference.Cluster)
			},
		},
		{
			desc: "config FailSafe Cluster strategy",
			opts: []ReferenceOption{
				WithClusterFailSafe(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyFailsafe, refOpts.Reference.Cluster)
			},
		},
		{
			desc: "config Forking Cluster strategy",
			opts: []ReferenceOption{
				WithClusterForking(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyForking, refOpts.Reference.Cluster)
			},
		},
		{
			desc: "config ZoneAware Cluster strategy",
			opts: []ReferenceOption{
				WithClusterZoneAware(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyZoneAware, refOpts.Reference.Cluster)
			},
		},
		{
			desc: "config AdaptiveService Cluster strategy",
			opts: []ReferenceOption{
				WithClusterAdaptiveService(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ClusterKeyAdaptiveService, refOpts.Reference.Cluster)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithLoadBalance(t *testing.T) {
	cases := []referenceOptionsInitCase{
		// todo(DMwangnima): think about default loadbalance strategy
		//{
		//	desc: "default Cluster strategy",
		//	opts: []ClientOption{},
		//	verify: func(t *testing.T, cli *Client, err error) {
		//		assert.Nil(t, err)
		//		assert.Equal(t, constant.ClusterKeyFailover, cli.cliOpts.overallReference.Cluster)
		//	},
		//},
		{
			desc: "config ConsistentHashing LoadBalance strategy",
			opts: []ReferenceOption{
				WithLoadBalanceConsistentHashing(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.LoadBalanceKeyConsistentHashing, refOpts.Reference.Loadbalance)
			},
		},
		{
			desc: "config LeastActive LoadBalance strategy",
			opts: []ReferenceOption{
				WithLoadBalanceLeastActive(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.LoadBalanceKeyLeastActive, refOpts.Reference.Loadbalance)
			},
		},
		{
			desc: "config Random LoadBalance strategy",
			opts: []ReferenceOption{
				WithLoadBalanceRandom(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.LoadBalanceKeyRandom, refOpts.Reference.Loadbalance)
			},
		},
		{
			desc: "config RoundRobin LoadBalance strategy",
			opts: []ReferenceOption{
				WithLoadBalanceRoundRobin(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.LoadBalanceKeyRoundRobin, refOpts.Reference.Loadbalance)
			},
		},
		{
			desc: "config P2C LoadBalance strategy",
			opts: []ReferenceOption{
				WithLoadBalanceP2C(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.LoadBalanceKeyP2C, refOpts.Reference.Loadbalance)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithRetries(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config normal retries",
			opts: []ReferenceOption{
				WithRetries(3),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "3", refOpts.Reference.Retries)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithGroup(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config normal group",
			opts: []ReferenceOption{
				WithGroup("test_group"),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "test_group", refOpts.Reference.Group)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithVersion(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config normal version",
			opts: []ReferenceOption{
				WithVersion("test_version"),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "test_version", refOpts.Reference.Version)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithSerialization(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "default Serialization",
			opts: []ReferenceOption{},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.ProtobufSerialization, refOpts.Reference.Serialization)
			},
		},
		{
			desc: "config JSON Serialization",
			opts: []ReferenceOption{
				WithSerializationJSON(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.JSONSerialization, refOpts.Reference.Serialization)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithProvidedBy(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config normal ProvidedBy",
			opts: []ReferenceOption{
				WithProvidedBy("test_instance"),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "test_instance", refOpts.Reference.ProvidedBy)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithParams(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config normal params",
			opts: []ReferenceOption{
				WithParams(map[string]string{
					"test_key": "test_val",
				}),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, map[string]string{"test_key": "test_val"}, refOpts.Reference.Params)
			},
		},
		{
			desc: "config nil params",
			opts: []ReferenceOption{
				WithParams(nil),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Nil(t, refOpts.Reference.Params)
			},
		},
		{
			desc: "config nil params with type information",
			opts: []ReferenceOption{
				WithParams((map[string]string)(nil)),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Nil(t, refOpts.Reference.Params)
			},
		},
		{
			desc: "config params without key-val",
			opts: []ReferenceOption{
				WithParams(map[string]string{}),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Nil(t, refOpts.Reference.Params)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

//func TestWithClientParam(t *testing.T) {
//	cases := []newClientCase{
//		{
//			desc: "config normal param",
//			opts: []ClientOption{
//				WithClientParam("test_key", "test_val"),
//			},
//			verify: func(t *testing.T, cli *Client, err error) {
//				assert.Nil(t, err)
//				assert.Equal(t, map[string]string{"test_key": "test_val"}, cli.cliOpts.overallReference.Params)
//			},
//		},
//		{
//			desc: "config normal param multiple times",
//			opts: []ClientOption{
//				WithClientParam("test_key", "test_val"),
//				WithClientParam("test_key1", "test_val1"),
//			},
//			verify: func(t *testing.T, cli *Client, err error) {
//				assert.Nil(t, err)
//				assert.Equal(t, map[string]string{"test_key": "test_val", "test_key1": "test_val1"}, cli.cliOpts.overallReference.Params)
//			},
//		},
//		{
//			desc: "config param with empty key",
//			opts: []ClientOption{
//				WithClientParam("", ""),
//			},
//			verify: func(t *testing.T, cli *Client, err error) {
//				assert.Nil(t, err)
//				assert.Equal(t, map[string]string{"": ""}, cli.cliOpts.overallReference.Params)
//			},
//		},
//	}
//	processNewClientCases(t, cases)
//}

func TestWithSticky(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config sticky",
			opts: []ReferenceOption{
				WithSticky(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.True(t, refOpts.Reference.Sticky)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithProtocol(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "default Protocol",
			opts: []ReferenceOption{},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "tri", refOpts.Reference.Protocol)
			},
		},
		{
			desc: "config Dubbo Protocol",
			opts: []ReferenceOption{
				WithProtocolDubbo(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, constant.Dubbo, refOpts.Reference.Protocol)
			},
		},
		{
			desc: "config Triple Protocol",
			opts: []ReferenceOption{
				WithProtocolTriple(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "tri", refOpts.Reference.Protocol)
			},
		},
		{
			desc: "config JsonRPC Protocol",
			opts: []ReferenceOption{
				WithProtocolJsonRPC(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "jsonrpc", refOpts.Reference.Protocol)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithRequestTimeout(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config normal RequestTimeout",
			opts: []ReferenceOption{
				WithRequestTimeout(6 * time.Second),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, "6s", refOpts.Reference.RequestTimeout)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithForceTag(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config ForceTag",
			opts: []ReferenceOption{
				WithForceTag(),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.True(t, refOpts.Reference.ForceTag)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}

func TestWithMeshProviderPort(t *testing.T) {
	cases := []referenceOptionsInitCase{
		{
			desc: "config normal MeshProviderPort",
			opts: []ReferenceOption{
				WithMeshProviderPort(20001),
			},
			verify: func(t *testing.T, refOpts *ReferenceOptions, err error) {
				assert.Nil(t, err)
				assert.Equal(t, 20001, refOpts.Reference.MeshProviderPort)
			},
		},
	}
	processReferenceOptionsInitCases(t, cases)
}
