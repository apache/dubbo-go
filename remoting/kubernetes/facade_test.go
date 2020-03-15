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
	"sync"
)
import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)
import (
	"github.com/apache/dubbo-go/common"
)

type mockFacade struct {
	client  *Client
	cltLock sync.Mutex
	wg      sync.WaitGroup
	URL     *common.URL
	done    chan struct{}
}

func (r *mockFacade) Client() *Client {
	return r.client
}

func (r *mockFacade) SetClient(client *Client) {
	r.client = client
}

func (r *mockFacade) ClientLock() *sync.Mutex {
	return &r.cltLock
}

func (r *mockFacade) WaitGroup() *sync.WaitGroup {
	return &r.wg
}

func (r *mockFacade) Done() chan struct{} {
	return r.done
}

func (r *mockFacade) GetUrl() common.URL {
	return *r.URL
}

func (r *mockFacade) Destroy() {
	close(r.done)
	r.wg.Wait()
}

func (r *mockFacade) RestartCallBack() bool {
	return true
}

func (r *mockFacade) IsAvailable() bool {
	return true
}

func (s *KubernetesClientTestSuite) Test_Facade() {

	t := s.T()

	mockClient, err := newMockClient(s.currentPod.GetNamespace(), func() (kubernetes.Interface, error) {

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

	url, _ := common.NewURL("mock://127.0.0.1")
	m := &mockFacade{
		client: mockClient,
		URL:    &url,
	}

	go HandleClientRestart(m)
	mockClient.Close()
}
