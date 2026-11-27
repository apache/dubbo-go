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

package consistenthashing

import (
	"fmt"
	"slices"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/suite"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

const (
	ip       = "192.168.1.0"
	port8080 = 8080
	port8081 = 8081

	url8080Short = "dubbo://192.168.1.0:8080"
	url8081Short = "dubbo://192.168.1.0:8081"
	url20000     = "dubbo://192.168.1.0:20000/org.apache.demo.HelloService?methods.echo.hash.arguments=0,1"
	url8080      = "dubbo://192.168.1.0:8080/org.apache.demo.HelloService?methods.echo.hash.arguments=0,1"
	url8081      = "dubbo://192.168.1.0:8081/org.apache.demo.HelloService?methods.echo.hash.arguments=0,1"
	url8082      = "dubbo://192.168.1.0:8082/org.apache.demo.HelloService?methods.echo.hash.arguments=0,1"
)

func TestConsistentHashSelectorSuite(t *testing.T) {
	suite.Run(t, new(consistentHashSelectorSuite))
}

type consistentHashSelectorSuite struct {
	suite.Suite
	selector *selector
}

func (s *consistentHashSelectorSuite) SetupTest() {
	var invokers []base.Invoker
	url, _ := common.NewURL(url20000)
	invokers = append(invokers, base.NewBaseInvoker(url))
	s.selector = newSelector(invokers, "echo", 999944)
}

func (s *consistentHashSelectorSuite) TestToKey() {
	result := s.selector.toKey([]any{"username", "age"})
	s.Equal("usernameage", result)
}

// TestToKeySingleNonZeroIndex verifies that a single non-zero hash.arguments
// index produces a key from the correct argument position.
func TestToKeyWithArgumentIndex(t *testing.T) {
	var invokers []base.Invoker
	// hash.arguments=1 means only the second argument (index 1) is used
	url, err := common.NewURL("dubbo://192.168.1.0:20000/org.apache.demo.HelloService?methods.echo.hash.arguments=1")
	if err != nil {
		t.Fatal(err)
	}
	invokers = append(invokers, base.NewBaseInvoker(url))
	sel := newSelector(invokers, "echo", 999944)

	key := sel.toKey([]any{"ignored", "used"})
	if key != "used" {
		t.Errorf("expected key %q, got %q", "used", key)
	}
}

func (s *consistentHashSelectorSuite) TestSelectForKey() {
	url1, _ := common.NewURL(url8080Short)
	url2, _ := common.NewURL(url8081Short)
	s.selector.virtualInvokers = make(map[uint32]base.Invoker)
	s.selector.virtualInvokers[99874] = base.NewBaseInvoker(url1)
	s.selector.virtualInvokers[9999945] = base.NewBaseInvoker(url2)
	s.selector.keys = []uint32{99874, 9999945}
	result := s.selector.selectForKey(9999944)
	s.Equal(url8081Short, result.GetURL().String())
}

func TestConsistentHashLoadBalanceSuite(t *testing.T) {
	suite.Run(t, new(consistentHashLoadBalanceSuite))
}

type consistentHashLoadBalanceSuite struct {
	suite.Suite
	url1     *common.URL
	url2     *common.URL
	url3     *common.URL
	invokers []base.Invoker
	invoker1 base.Invoker
	invoker2 base.Invoker
	invoker3 base.Invoker
	lb       loadbalance.LoadBalance
}

func (s *consistentHashLoadBalanceSuite) SetupTest() {
	var err error
	s.url1, err = common.NewURL(url8080)
	s.Require().NoError(err)
	s.url2, err = common.NewURL(url8081)
	s.Require().NoError(err)
	s.url3, err = common.NewURL(url8082)
	s.Require().NoError(err)

	s.invoker1 = base.NewBaseInvoker(s.url1)
	s.invoker2 = base.NewBaseInvoker(s.url2)
	s.invoker3 = base.NewBaseInvoker(s.url3)

	s.invokers = append(s.invokers, s.invoker1, s.invoker2, s.invoker3)
	s.lb = newConshashLoadBalance()
}

func (s *consistentHashLoadBalanceSuite) TestSelect() {
	args := []any{"name", "password", "age"}
	invoker := s.lb.Select(s.invokers, invocation.NewRPCInvocation("echo", args, nil))
	s.Equal(fmt.Sprintf("%s:%d", ip, port8081), invoker.GetURL().Location)

	args = []any{"ok", "abc"}
	invoker = s.lb.Select(s.invokers, invocation.NewRPCInvocation("echo", args, nil))
	s.Equal(fmt.Sprintf("%s:%d", ip, port8080), invoker.GetURL().Location)
}

// TestConcurrentSelect reproduces the race condition on the global `selectors` map.
// Run with: go test -race -run TestConsistentHashLoadBalanceSuite/TestConcurrentSelect -count=1 ./cluster/loadbalance/consistenthashing/
func (s *consistentHashLoadBalanceSuite) TestConcurrentSelect() {
	const goroutines = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			args := []any{fmt.Sprintf("user-%d", id), "password"}
			invoker := s.lb.Select(s.invokers, invocation.NewRPCInvocation("echo", args, nil))
			s.NotNilf(invoker, "Select returned nil for goroutine %d", id)
		}(i)
	}
	wg.Wait()
}

// TestSelectAfterProviderReplaced replaces one provider with another, which keeps the number of
// invokers the same. Every key must then go to an invoker in the current list.
func (s *consistentHashLoadBalanceSuite) TestSelectAfterProviderReplaced() {
	url, err := common.NewURL("dubbo://192.168.1.0:8083/org.apache.demo.HelloService?methods.echo.hash.arguments=0,1")
	s.Require().NoError(err)
	replacement := base.NewBaseInvoker(url)

	previous := []base.Invoker{s.invoker1, s.invoker2, s.invoker3}
	for i := range 64 {
		s.lb.Select(previous, invocation.NewRPCInvocation("echo", []any{fmt.Sprint(i), "p"}, nil))
	}
	current := []base.Invoker{replacement, s.invoker2, s.invoker3}
	for i := range 64 {
		invoker := s.lb.Select(current, invocation.NewRPCInvocation("echo", []any{fmt.Sprint(i), "p"}, nil))
		s.Truef(slices.Contains(current, invoker), "key %d went to %s, which is not in the invoker list",
			i, invoker.GetURL().Location)
	}
}

// TestSelectFromEqualSizedSubsets alternates between two disjoint invoker lists of the same size,
// as a router does when it gives different requests different providers.
func (s *consistentHashLoadBalanceSuite) TestSelectFromEqualSizedSubsets() {
	url, err := common.NewURL("dubbo://192.168.1.0:8083/org.apache.demo.HelloService?methods.echo.hash.arguments=0,1")
	s.Require().NoError(err)
	first := []base.Invoker{s.invoker1, s.invoker3}
	second := []base.Invoker{s.invoker2, base.NewBaseInvoker(url)}

	for i := range 64 {
		for _, invokers := range [][]base.Invoker{first, second} {
			invoker := s.lb.Select(invokers, invocation.NewRPCInvocation("echo", []any{fmt.Sprint(i), "p"}, nil))
			s.Truef(slices.Contains(invokers, invoker), "key %d went to %s, which is not in the invoker list",
				i, invoker.GetURL().Location)
		}
	}
}

// TestSelectKeepsRingForSameInvokers checks that an unchanged invoker list reuses the cached ring.
func (s *consistentHashLoadBalanceSuite) TestSelectKeepsRingForSameInvokers() {
	invokers := []base.Invoker{s.invoker1, s.invoker2, s.invoker3}
	inv := invocation.NewRPCInvocation("echo", []any{"name", "password"}, nil)
	first := s.lb.Select(invokers, inv)
	key := s.invoker1.GetURL().ServiceKey() + ".echo"
	selectorsMu.RLock()
	cached := selectors[key]
	selectorsMu.RUnlock()

	again := s.lb.Select(slices.Clone(invokers), inv)
	selectorsMu.RLock()
	defer selectorsMu.RUnlock()
	s.Same(cached, selectors[key])
	s.Same(first, again)
}
